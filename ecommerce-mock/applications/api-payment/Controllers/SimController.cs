// Controllers/SimController.cs
// ─────────────────────────────────────────────────────────────────────────────
// Simulation endpoints that intentionally trigger 5xx responses.
//
// Endpoints
//   GET    /sim/bad-column   → SELECT not_existed FROM Payments → 500
//   POST   /sim/bad-insert   → INSERT INTO Payments (not_existed) → 500
//   DELETE /sim/bad-delete   → DELETE FROM Payments WHERE not_existed → 500

using ApiPayment.Data;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;
using Serilog.Context;

namespace ApiPayment.Controllers;

[ApiController]
[Route("sim")]
public class SimController(AppDbContext db, ILogger<SimController> logger) : ControllerBase
{
    // ── GET /sim/bad-column ───────────────────────────────────────────────────
    // Runs raw SQL referencing a column that does not exist.
    // MySQL returns:
    //   Unknown column 'not_existed' in 'field list' (Error 1054)
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
            cmd.CommandText = "SELECT not_existed FROM Payments LIMIT 1";
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

    // ── POST /sim/bad-insert ──────────────────────────────────────────────────
    // Runs an INSERT referencing a column that does not exist.
    // MySQL returns:
    //   Unknown column 'not_existed' in 'field list' (Error 1054)
    [HttpPost("bad-insert")]
    public async Task<IActionResult> BadInsert()
    {
        var requestId = HttpContext.TraceIdentifier;

        using (LogContext.PushProperty("Category", "SIM"))
        using (LogContext.PushProperty("RequestId", requestId))
        {
            logger.LogInformation("sim: bad-insert triggered");
        }

        try
        {
            var conn = db.Database.GetDbConnection();
            await conn.OpenAsync();
            await using var cmd = conn.CreateCommand();
            cmd.CommandText = "INSERT INTO Payments (not_existed) VALUES ('sim')";
            await cmd.ExecuteNonQueryAsync();

            // Should never reach here.
            using (LogContext.PushProperty("Category", "SIM"))
                logger.LogWarning("sim: bad-insert unexpectedly succeeded");

            return Ok(new { sim = "bad-insert", result = "unexpected_success" });
        }
        catch (Exception ex)
        {
            using (LogContext.PushProperty("Category", "DB_ERROR"))
            using (LogContext.PushProperty("RequestId", requestId))
            {
                logger.LogError(ex, "sim: insert failed as expected");
            }

            return StatusCode(500, new
            {
                error    = "database error",
                sim      = "bad-insert",
                detail   = ex.Message,
                category = "DB_ERROR",
            });
        }
    }

    // ── DELETE /sim/bad-delete ────────────────────────────────────────────────
    // Runs a DELETE referencing a column that does not exist.
    // MySQL returns:
    //   Unknown column 'not_existed' in 'where clause' (Error 1054)
    [HttpDelete("bad-delete")]
    public async Task<IActionResult> BadDelete()
    {
        var requestId = HttpContext.TraceIdentifier;

        using (LogContext.PushProperty("Category", "SIM"))
        using (LogContext.PushProperty("RequestId", requestId))
        {
            logger.LogInformation("sim: bad-delete triggered");
        }

        try
        {
            var conn = db.Database.GetDbConnection();
            await conn.OpenAsync();
            await using var cmd = conn.CreateCommand();
            cmd.CommandText = "DELETE FROM Payments WHERE not_existed = 'sim'";
            await cmd.ExecuteNonQueryAsync();

            // Should never reach here.
            using (LogContext.PushProperty("Category", "SIM"))
                logger.LogWarning("sim: bad-delete unexpectedly succeeded");

            return Ok(new { sim = "bad-delete", result = "unexpected_success" });
        }
        catch (Exception ex)
        {
            using (LogContext.PushProperty("Category", "DB_ERROR"))
            using (LogContext.PushProperty("RequestId", requestId))
            {
                logger.LogError(ex, "sim: delete failed as expected");
            }

            return StatusCode(500, new
            {
                error    = "database error",
                sim      = "bad-delete",
                detail   = ex.Message,
                category = "DB_ERROR",
            });
        }
    }
}
