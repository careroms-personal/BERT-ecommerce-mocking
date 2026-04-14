using System.ComponentModel.DataAnnotations;
using System.ComponentModel.DataAnnotations.Schema;

namespace ApiPayment.Models;

[Table("payments")]
public class Payment
{
    [Key]
    [Column("id")]
    public Guid Id { get; set; } = Guid.NewGuid();

    [Required]
    [Column("order_id")]
    public Guid OrderId { get; set; }

    [Required]
    [Column("customer_id")]
    public Guid CustomerId { get; set; }

    [Required]
    [Column("amount")]
    public decimal Amount { get; set; }

    [Required]
    [Column("status")]
    [MaxLength(20)]
    public string Status { get; set; } = PaymentStatus.Pending;

    [Column("provider_ref")]
    [MaxLength(100)]
    public string? ProviderRef { get; set; }

    [Column("failure_reason")]
    [MaxLength(100)]
    public string? FailureReason { get; set; }

    [Column("retry_count")]
    public int RetryCount { get; set; } = 0;

    [Column("created_at")]
    public DateTime CreatedAt { get; set; } = DateTime.UtcNow;

    [Column("updated_at")]
    public DateTime UpdatedAt { get; set; } = DateTime.UtcNow;
}

public static class PaymentStatus
{
    public const string Pending    = "PENDING";
    public const string Processing = "PROCESSING";
    public const string Success    = "SUCCESS";
    public const string Failed     = "FAILED";
    public const string Refunded   = "REFUNDED";
}
