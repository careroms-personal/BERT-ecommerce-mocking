using System.ComponentModel.DataAnnotations;
using System.ComponentModel.DataAnnotations.Schema;

namespace ApiCustomer.Models;

[Table("revoked_tokens")]
public class RevokedToken
{
    [Key]
    [Column("jti")]
    public Guid Jti { get; set; }

    [Column("customer_id")]
    public Guid CustomerId { get; set; }

    [Column("revoked_at")]
    public DateTime RevokedAt { get; set; } = DateTime.UtcNow;

    [Column("expires_at")]
    public DateTime ExpiresAt { get; set; }
}
