using ApiCustomer.Data;
using ApiCustomer.Services;
using Microsoft.AspNetCore.Authentication.JwtBearer;
using Microsoft.EntityFrameworkCore;
using Microsoft.IdentityModel.Tokens;
using Serilog;
using Serilog.Context;
using Serilog.Formatting.Compact;
using System.Text;

// Bootstrap logger for startup logs before host is built
Log.Logger = new LoggerConfiguration()
    .Enrich.WithProperty("Service", "api-customer")
    .Enrich.WithProperty("Category", "SYSTEM")
    .WriteTo.Console(new CompactJsonFormatter())
    .CreateBootstrapLogger();

try
{
    Log.Information("Service starting {@Meta}", new { Service = "api-customer", Version = "1.0.0" });

    var builder = WebApplication.CreateBuilder(args);

    // ── Serilog ──────────────────────────────────────────────────────────────
    builder.Host.UseSerilog((ctx, services, config) =>
    {
        config
            .ReadFrom.Configuration(ctx.Configuration)
            .ReadFrom.Services(services)
            .Enrich.FromLogContext()
            .Enrich.WithProperty("Service", "api-customer");
    });

    // ── Database ─────────────────────────────────────────────────────────────
    builder.Services.AddDbContext<AppDbContext>(opts =>
        opts.UseNpgsql(builder.Configuration.GetConnectionString("DefaultConnection")));

    // ── JWT auth ─────────────────────────────────────────────────────────────
    var jwtSecret = builder.Configuration["Jwt:Secret"]
        ?? throw new InvalidOperationException("Jwt:Secret not configured");

    builder.Services.AddAuthentication(JwtBearerDefaults.AuthenticationScheme)
        .AddJwtBearer(opts =>
        {
            opts.TokenValidationParameters = new TokenValidationParameters
            {
                ValidateIssuer = false,
                ValidateAudience = false,
                ValidateLifetime = true,
                ValidateIssuerSigningKey = true,
                IssuerSigningKey = new SymmetricSecurityKey(Encoding.UTF8.GetBytes(jwtSecret)),
                ClockSkew = TimeSpan.Zero
            };
        });

    builder.Services.AddAuthorization();
    builder.Services.AddControllers();
    builder.Services.AddSingleton<TokenService>();

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
                404 => "NOT_FOUND",
                >= 500 => "ERROR",
                _ => "AUTH"
            });
            diag.Set("ClientIp", ctx.Connection.RemoteIpAddress?.ToString());
        };
    });

    app.UseAuthentication();
    app.UseAuthorization();
    app.MapControllers();

    app.MapGet("/health", () =>
    {
        using (LogContext.PushProperty("Category", "SYSTEM"))
        {
            Log.Information("Health check ok");
        }
        return Results.Ok(new { status = "ok", service = "api-customer" });
    });

    using (LogContext.PushProperty("Category", "SYSTEM"))
    {
        Log.Information("Server listening on port {Port}", builder.Configuration["ASPNETCORE_URLS"] ?? "8082");
    }

    app.Run();
}
catch (Exception ex)
{
    using (LogContext.PushProperty("Category", "ERROR"))
    {
        Log.Fatal(ex, "Service terminated unexpectedly");
    }
}
finally
{
    Log.CloseAndFlush();
}
