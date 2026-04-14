using System.ComponentModel.DataAnnotations;

namespace ApiPayment.Models;

public record ProcessPaymentRequest(
    [Required] Guid OrderId,
    [Required] Guid CustomerId,
    [Required][Range(0.01, double.MaxValue)] decimal Amount
);

public record WebhookPayload(
    [Required] string ProviderRef,
    [Required] string Status,   // SUCCESS | FAILED
    string? FailureReason
);

public record ProviderResult(
    bool Success,
    string? ProviderRef,
    string? FailureReason,
    bool TimedOut = false
);
