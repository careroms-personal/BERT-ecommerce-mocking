using ApiPayment.Data;
using ApiPayment.Services;
using Microsoft.EntityFrameworkCore;
using Serilog;
using Serilog.Context;
using Serilog.Formatting.Json;

Log.Logger = new LoggerConfiguration()
    .Enrich.WithProperty("Service", "api-payment")
    .Enrich.WithProperty("Category", "SYSTEM")
    .WriteTo.Console(new JsonFormatter())
    .CreateBootstrapLogger();

try
{
    Log.Information("Service starting {@Meta}", new { Service = "api-payment", Version = "1.0.0" });

    var builder = WebApplication.CreateBuilder(args);

    // ── Serilog ──────────────────────────────────────────────────────────────
    builder.Host.UseSerilog((ctx, services, config) =>
    {
        config
            .ReadFrom.Configuration(ctx.Configuration)
            .ReadFrom.Services(services)
            .Enrich.FromLogContext()
            .Enrich.WithProperty("Service", "api-payment");
    });

    // ── Database (MySQL via Pomelo) ───────────────────────────────────────────
    var connStr = builder.Configuration.GetConnectionString("DefaultConnection")!;
    builder.Services.AddDbContext<AppDbContext>(opts =>
        opts.UseMySql(connStr, ServerVersion.AutoDetect(connStr)));

    builder.Services.AddControllers();
    builder.Services.AddSingleton<MockPaymentProvider>();

    var app = builder.Build();

    // ── DB migration on startup ───────────────────────────────────────────────
    using (var scope = app.Services.CreateScope())
    {
        var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();
        try
        {
            using (LogContext.PushProperty("Category", "SYSTEM"))
            {
                Log.Information("Running database migration");
                db.Database.EnsureCreated();
                Log.Information("Database migration complete");
            }
        }
        catch (Exception ex)
        {
            using (LogContext.PushProperty("Category", "DB_ERROR"))
                Log.Error(ex, "Database migration failed, continuing without schema — requests may fail");
        }
    }

    // ── Middleware ────────────────────────────────────────────────────────────
    app.UseSerilogRequestLogging(opts =>
    {
        opts.EnrichDiagnosticContext = (diag, ctx) =>
        {
            diag.Set("Category", ctx.Response.StatusCode switch
            {
                402 => "PAYMENT_FAIL",
                404 => "NOT_FOUND",
                429 => "RETRY_EXHAUSTED",
                >= 500 => "ERROR",
                _ => "PAYMENT"
            });
            diag.Set("ClientIp", ctx.Connection.RemoteIpAddress?.ToString());
        };
    });

    app.MapControllers();

    app.MapGet("/health", () =>
    {
        using (LogContext.PushProperty("Category", "SYSTEM"))
            Log.Information("Health check ok");
        return Results.Ok(new { status = "ok", service = "api-payment" });
    });

    using (LogContext.PushProperty("Category", "SYSTEM"))
        Log.Information("Server listening on port 8085");

    app.Run();
}
catch (Exception ex)
{
    using (LogContext.PushProperty("Category", "ERROR"))
        Log.Fatal(ex, "Service terminated unexpectedly");
}
finally
{
    Log.CloseAndFlush();
}
