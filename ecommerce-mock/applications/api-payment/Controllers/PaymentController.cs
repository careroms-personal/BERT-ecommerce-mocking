using ApiPayment.Data;
using ApiPayment.Models;
using ApiPayment.Services;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;
using Serilog.Context;

namespace ApiPayment.Controllers;

[ApiController]
[Route("payments")]
public class PaymentController(
    AppDbContext db,
    MockPaymentProvider provider,
    IConfiguration config,
    ILogger<PaymentController> logger) : ControllerBase
{
    private readonly int _maxRetries = int.Parse(config["Payment:MaxRetries"] ?? "3");
    private readonly int _providerTimeoutMs = int.Parse(config["Payment:ProviderTimeoutMs"] ?? "4000");

    // ── POST /payments ────────────────────────────────────────────────────────
    [HttpPost]
    public async Task<IActionResult> ProcessPayment([FromBody] ProcessPaymentRequest req)
    {
        var requestId = HttpContext.TraceIdentifier;

        using (LogContext.PushProperty("Category", "PAYMENT"))
        using (LogContext.PushProperty("RequestId", requestId))
        using (LogContext.PushProperty("OrderId", req.OrderId))
        using (LogContext.PushProperty("CustomerId", req.CustomerId))
        using (LogContext.PushProperty("Amount", req.Amount))
        {
            logger.LogInformation("Processing payment for order {OrderId} amount {Amount}", req.OrderId, req.Amount);

            var payment = new Payment
            {
                OrderId    = req.OrderId,
                CustomerId = req.CustomerId,
                Amount     = req.Amount,
                Status     = PaymentStatus.Processing,
            };
            db.Payments.Add(payment);

            try { await db.SaveChangesAsync(); }
            catch (Exception ex)
            {
                using (LogContext.PushProperty("Category", "DB_ERROR"))
                    logger.LogError(ex, "Failed to create payment record for order {OrderId}", req.OrderId);
                return StatusCode(500, new { error = "database error" });
            }

            using (LogContext.PushProperty("PaymentId", payment.Id))
            {
                logger.LogInformation("Payment record created, calling provider");
            }

            // Call mock provider with timeout
            using var cts = new CancellationTokenSource(TimeSpan.FromMilliseconds(_providerTimeoutMs));
            ProviderResult result;
            try { result = await provider.ChargeAsync(req.Amount, cts.Token); }
            catch (OperationCanceledException)
            {
                result = new ProviderResult(false, null, "provider_timeout", TimedOut: true);
            }

            return await FinalisePayment(payment, result, requestId);
        }
    }

    // ── GET /payments/:id ─────────────────────────────────────────────────────
    [HttpGet("{id:guid}")]
    public async Task<IActionResult> GetPayment(Guid id)
    {
        var requestId = HttpContext.TraceIdentifier;

        using (LogContext.PushProperty("Category", "PAYMENT"))
        using (LogContext.PushProperty("RequestId", requestId))
        using (LogContext.PushProperty("PaymentId", id))
        {
            logger.LogInformation("Fetching payment status {PaymentId}", id);

            var payment = await db.Payments.FindAsync(id);
            if (payment is null)
            {
                using (LogContext.PushProperty("Category", "NOT_FOUND"))
                    logger.LogWarning("Payment not found {PaymentId}", id);
                return NotFound(new { error = "payment not found", payment_id = id });
            }

            logger.LogInformation("Payment status {PaymentId} is {Status}", id, payment.Status);
            return Ok(payment);
        }
    }

    // ── POST /payments/:id/retry ──────────────────────────────────────────────
    [HttpPost("{id:guid}/retry")]
    public async Task<IActionResult> RetryPayment(Guid id)
    {
        var requestId = HttpContext.TraceIdentifier;

        using (LogContext.PushProperty("Category", "PAYMENT"))
        using (LogContext.PushProperty("RequestId", requestId))
        using (LogContext.PushProperty("PaymentId", id))
        {
            logger.LogInformation("Retry requested for payment {PaymentId}", id);

            var payment = await db.Payments.FindAsync(id);
            if (payment is null)
            {
                using (LogContext.PushProperty("Category", "NOT_FOUND"))
                    logger.LogWarning("Payment not found for retry {PaymentId}", id);
                return NotFound(new { error = "payment not found", payment_id = id });
            }

            if (payment.Status != PaymentStatus.Failed)
            {
                using (LogContext.PushProperty("Category", "ERROR"))
                    logger.LogWarning("Retry rejected — payment is not in FAILED state {PaymentId} {Status}", id, payment.Status);
                return BadRequest(new { error = "only FAILED payments can be retried", status = payment.Status });
            }

            if (payment.RetryCount >= _maxRetries)
            {
                using (LogContext.PushProperty("Category", "RETRY_EXHAUSTED"))
                using (LogContext.PushProperty("RetryCount", payment.RetryCount))
                using (LogContext.PushProperty("MaxRetries", _maxRetries))
                {
                    logger.LogWarning(
                        "Retry exhausted for payment {PaymentId} — {RetryCount}/{MaxRetries} attempts used",
                        id, payment.RetryCount, _maxRetries);
                }
                return StatusCode(429, new { error = "max retries exceeded", retry_count = payment.RetryCount, max_retries = _maxRetries });
            }

            payment.RetryCount++;
            payment.Status = PaymentStatus.Processing;
            payment.UpdatedAt = DateTime.UtcNow;

            using (LogContext.PushProperty("RetryCount", payment.RetryCount))
                logger.LogInformation("Retrying payment {PaymentId} attempt {RetryCount}/{MaxRetries}", id, payment.RetryCount, _maxRetries);

            try { await db.SaveChangesAsync(); }
            catch (Exception ex)
            {
                using (LogContext.PushProperty("Category", "DB_ERROR"))
                    logger.LogError(ex, "Failed to update payment for retry {PaymentId}", id);
                return StatusCode(500, new { error = "database error" });
            }

            using var cts = new CancellationTokenSource(TimeSpan.FromMilliseconds(_providerTimeoutMs));
            ProviderResult result;
            try { result = await provider.ChargeAsync(payment.Amount, cts.Token); }
            catch (OperationCanceledException)
            {
                result = new ProviderResult(false, null, "provider_timeout", TimedOut: true);
            }

            return await FinalisePayment(payment, result, requestId);
        }
    }

    // ── POST /payments/webhook ────────────────────────────────────────────────
    [HttpPost("webhook")]
    public async Task<IActionResult> Webhook([FromBody] WebhookPayload payload)
    {
        var requestId = HttpContext.TraceIdentifier;

        using (LogContext.PushProperty("Category", "PAYMENT"))
        using (LogContext.PushProperty("RequestId", requestId))
        using (LogContext.PushProperty("ProviderRef", payload.ProviderRef))
        {
            logger.LogInformation("Webhook received from provider ref {ProviderRef} status {Status}", payload.ProviderRef, payload.Status);

            var payment = await db.Payments
                .FirstOrDefaultAsync(p => p.ProviderRef == payload.ProviderRef);

            if (payment is null)
            {
                using (LogContext.PushProperty("Category", "NOT_FOUND"))
                    logger.LogWarning("Webhook — no payment found for provider ref {ProviderRef}", payload.ProviderRef);
                return NotFound(new { error = "payment not found for provider ref" });
            }

            payment.Status    = payload.Status == "SUCCESS" ? PaymentStatus.Success : PaymentStatus.Failed;
            payment.UpdatedAt = DateTime.UtcNow;
            if (payload.FailureReason is not null)
                payment.FailureReason = payload.FailureReason;

            try { await db.SaveChangesAsync(); }
            catch (Exception ex)
            {
                using (LogContext.PushProperty("Category", "DB_ERROR"))
                    logger.LogError(ex, "Failed to update payment from webhook {ProviderRef}", payload.ProviderRef);
                return StatusCode(500, new { error = "database error" });
            }

            using (LogContext.PushProperty("PaymentId", payment.Id))
                logger.LogInformation("Payment {PaymentId} updated via webhook to {Status}", payment.Id, payment.Status);

            return Ok(new { message = "webhook processed", payment_id = payment.Id, status = payment.Status });
        }
    }

    // ── Shared finalise helper ────────────────────────────────────────────────
    private async Task<IActionResult> FinalisePayment(Payment payment, ProviderResult result, string requestId)
    {
        if (result.TimedOut)
        {
            payment.Status        = PaymentStatus.Failed;
            payment.FailureReason = "provider_timeout";
            payment.UpdatedAt     = DateTime.UtcNow;

            using (LogContext.PushProperty("Category", "PAYMENT_TIMEOUT"))
            using (LogContext.PushProperty("PaymentId", payment.Id))
            using (LogContext.PushProperty("RequestId", requestId))
                logger.LogWarning("Payment timed out waiting for provider {PaymentId} order {OrderId}", payment.Id, payment.OrderId);
        }
        else if (result.Success)
        {
            payment.Status      = PaymentStatus.Success;
            payment.ProviderRef = result.ProviderRef;
            payment.UpdatedAt   = DateTime.UtcNow;

            using (LogContext.PushProperty("Category", "PAYMENT"))
            using (LogContext.PushProperty("PaymentId", payment.Id))
            using (LogContext.PushProperty("ProviderRef", result.ProviderRef))
            using (LogContext.PushProperty("RequestId", requestId))
                logger.LogInformation("Payment successful {PaymentId} order {OrderId} amount {Amount}", payment.Id, payment.OrderId, payment.Amount);
        }
        else
        {
            payment.Status        = PaymentStatus.Failed;
            payment.FailureReason = result.FailureReason;
            payment.UpdatedAt     = DateTime.UtcNow;

            using (LogContext.PushProperty("Category", "PAYMENT_FAIL"))
            using (LogContext.PushProperty("PaymentId", payment.Id))
            using (LogContext.PushProperty("FailureReason", result.FailureReason))
            using (LogContext.PushProperty("RequestId", requestId))
                logger.LogWarning("Payment declined {PaymentId} order {OrderId} reason {FailureReason}", payment.Id, payment.OrderId, result.FailureReason);
        }

        try { await db.SaveChangesAsync(); }
        catch (Exception ex)
        {
            using (LogContext.PushProperty("Category", "DB_ERROR"))
                logger.LogError(ex, "Failed to finalise payment record {PaymentId}", payment.Id);
            return StatusCode(500, new { error = "database error" });
        }

        var statusCode = payment.Status == PaymentStatus.Success ? 200 : 402;
        return StatusCode(statusCode, payment);
    }
}
