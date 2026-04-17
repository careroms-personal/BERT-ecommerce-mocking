// Controllers/SimController.cs
// ─────────────────────────────────────────────────────────────────────────────
// Simulation endpoints that intentionally trigger 5xx responses.
//
// Endpoints
//   GET /sim/bad-column   → SELECT not_existed FROM customers → 500

using ApiCustomer.Data;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;
using Serilog.Context;

namespace ApiCustomer.Controllers;

[ApiController]
[Route("sim")]
public class SimController(AppDbContext db, ILogger<SimController> logger) : ControllerBase
{
    // ── GET /sim/bad-column ───────────────────────────────────────────────────
    // Runs raw SQL referencing a column that does not exist.
    // PostgreSQL returns:
    //   ERROR: column "not_existed" does not exist (SQLSTATE 42703)
    [HttpGet("bad-column")]
    public async Task<IActionResult> BadColumn()
    {
        var requestId = HttpContext.TraceIdentifier;

        using (LogContext.PushProperty("Category", "SIM"))
        using (LogContext.PushProperty("RequestId", requestId))
        {
            logger.LogInformation("sim: bad-column query triggered");
        }

        try
        {
            var conn = db.Database.GetDbConnection();
            await conn.OpenAsync();
            await using var cmd = conn.CreateCommand();
            cmd.CommandText = "SELECT not_existed FROM customers LIMIT 1";
            await cmd.ExecuteReaderAsync();

            // Should never reach here.
            using (LogContext.PushProperty("Category", "SIM"))
                logger.LogWarning("sim: bad-column query unexpectedly succeeded");

            return Ok(new { sim = "bad-column", result = "unexpected_success" });
        }
        catch (Exception ex)
        {
            using (LogContext.PushProperty("Category", "DB_ERROR"))
            using (LogContext.PushProperty("RequestId", requestId))
            {
                logger.LogError(ex, "sim: query failed as expected");
            }

            return StatusCode(500, new
            {
                error    = "database error",
                sim      = "bad-column",
                detail   = ex.Message,
                category = "DB_ERROR",
            });
        }
    }
}
