using ApiCustomer.Data;
using ApiCustomer.Models;
using ApiCustomer.Services;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;
using Serilog.Context;

namespace ApiCustomer.Controllers;

[ApiController]
[Route("auth")]
public class AuthController(AppDbContext db, TokenService tokenService, ILogger<AuthController> logger) : ControllerBase
{
    [HttpPost("login")]
    public async Task<IActionResult> Login([FromBody] LoginRequest req)
    {
        var requestId = HttpContext.TraceIdentifier;

        using (LogContext.PushProperty("Category", "AUTH"))
        using (LogContext.PushProperty("RequestId", requestId))
        using (LogContext.PushProperty("Email", req.Email))
        {
            logger.LogInformation("Login attempt for {Email}", req.Email);

            var customer = await db.Customers
                .FirstOrDefaultAsync(c => c.Email == req.Email);

            if (customer is null)
            {
                using (LogContext.PushProperty("Category", "AUTH_FAIL"))
                {
                    logger.LogWarning("Login failed — email not found {Email}", req.Email);
                }
                return Unauthorized(new { error = "invalid credentials" });
            }

            if (!BCrypt.Net.BCrypt.Verify(req.Password, customer.PasswordHash))
            {
                using (LogContext.PushProperty("Category", "AUTH_FAIL"))
                using (LogContext.PushProperty("CustomerId", customer.Id))
                {
                    logger.LogWarning("Login failed — wrong password for {Email}", req.Email);
                }
                return Unauthorized(new { error = "invalid credentials" });
            }

            var (accessToken, accessJti, expiresAt) = tokenService.GenerateAccessToken(customer);
            var (refreshToken, refreshJti, refreshExpiresAt) = tokenService.GenerateRefreshToken(customer);

            using (LogContext.PushProperty("CustomerId", customer.Id))
            {
                logger.LogInformation("Login successful for {Email}", req.Email);
            }

            return Ok(new TokenResponse(accessToken, refreshToken, expiresAt, CustomerMapper.ToDto(customer)));
        }
    }

    [HttpPost("logout")]
    public async Task<IActionResult> Logout()
    {
        var requestId = HttpContext.TraceIdentifier;
        var authHeader = Request.Headers.Authorization.FirstOrDefault();
        var token = authHeader?.Replace("Bearer ", "") ?? string.Empty;

        using (LogContext.PushProperty("Category", "AUTH"))
        using (LogContext.PushProperty("RequestId", requestId))
        {
            logger.LogInformation("Logout requested");

            var principal = tokenService.ValidateToken(token, out var jti, out var expired);

            if (expired)
            {
                using (LogContext.PushProperty("Category", "AUTH_EXPIRED"))
                {
                    logger.LogWarning("Logout with expired token {Jti}", jti);
                }
                return Unauthorized(new { error = "token already expired" });
            }

            if (principal is null || jti == Guid.Empty)
            {
                using (LogContext.PushProperty("Category", "AUTH_FAIL"))
                {
                    logger.LogWarning("Logout with invalid token");
                }
                return Unauthorized(new { error = "invalid token" });
            }

            var customerIdStr = principal.FindFirst(System.IdentityModel.Tokens.Jwt.JwtRegisteredClaimNames.Sub)?.Value;
            Guid.TryParse(customerIdStr, out var customerId);

            var alreadyRevoked = await db.RevokedTokens.AnyAsync(t => t.Jti == jti);
            if (!alreadyRevoked)
            {
                db.RevokedTokens.Add(new RevokedToken
                {
                    Jti = jti,
                    CustomerId = customerId,
                    ExpiresAt = DateTime.UtcNow.AddHours(1)
                });
                await db.SaveChangesAsync();
            }

            using (LogContext.PushProperty("CustomerId", customerId))
            using (LogContext.PushProperty("Jti", jti))
            {
                logger.LogInformation("Token revoked, logout successful");
            }

            return Ok(new { message = "logged out" });
        }
    }

    [HttpPost("refresh")]
    public async Task<IActionResult> Refresh()
    {
        var requestId = HttpContext.TraceIdentifier;
        var authHeader = Request.Headers.Authorization.FirstOrDefault();
        var token = authHeader?.Replace("Bearer ", "") ?? string.Empty;

        using (LogContext.PushProperty("Category", "AUTH"))
        using (LogContext.PushProperty("RequestId", requestId))
        {
            logger.LogInformation("Token refresh requested");

            var principal = tokenService.ValidateToken(token, out var jti, out var expired);

            if (expired)
            {
                using (LogContext.PushProperty("Category", "AUTH_EXPIRED"))
                {
                    logger.LogWarning("Refresh token expired {Jti}", jti);
                }
                return Unauthorized(new { error = "refresh token expired, please login again" });
            }

            if (principal is null)
            {
                using (LogContext.PushProperty("Category", "AUTH_EXPIRED"))
                {
                    logger.LogWarning("Invalid refresh token");
                }
                return Unauthorized(new { error = "invalid token" });
            }

            var isRevoked = await db.RevokedTokens.AnyAsync(t => t.Jti == jti);
            if (isRevoked)
            {
                using (LogContext.PushProperty("Category", "AUTH_EXPIRED"))
                using (LogContext.PushProperty("Jti", jti))
                {
                    logger.LogWarning("Attempted use of revoked token {Jti}", jti);
                }
                return Unauthorized(new { error = "token has been revoked" });
            }

            var customerIdStr = principal.FindFirst(System.IdentityModel.Tokens.Jwt.JwtRegisteredClaimNames.Sub)?.Value;
            if (!Guid.TryParse(customerIdStr, out var customerId))
                return Unauthorized(new { error = "invalid token claims" });

            var customer = await db.Customers.FindAsync(customerId);
            if (customer is null)
            {
                using (LogContext.PushProperty("Category", "NOT_FOUND"))
                {
                    logger.LogWarning("Customer not found during token refresh {CustomerId}", customerId);
                }
                return Unauthorized(new { error = "customer not found" });
            }

            // Revoke old token
            db.RevokedTokens.Add(new RevokedToken { Jti = jti, CustomerId = customerId, ExpiresAt = DateTime.UtcNow.AddDays(7) });
            await db.SaveChangesAsync();

            var (newAccess, newAccessJti, expiresAt) = tokenService.GenerateAccessToken(customer);
            var (newRefresh, newRefreshJti, _) = tokenService.GenerateRefreshToken(customer);

            using (LogContext.PushProperty("CustomerId", customerId))
            {
                logger.LogInformation("Token refreshed for {Email}", customer.Email);
            }

            return Ok(new TokenResponse(newAccess, newRefresh, expiresAt, CustomerMapper.ToDto(customer)));
        }
    }
}
