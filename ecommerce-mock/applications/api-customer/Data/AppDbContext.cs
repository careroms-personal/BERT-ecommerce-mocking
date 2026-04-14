using ApiCustomer.Models;
using Microsoft.EntityFrameworkCore;

namespace ApiCustomer.Data;

public class AppDbContext(DbContextOptions<AppDbContext> options) : DbContext(options)
{
    public DbSet<Customer> Customers => Set<Customer>();
    public DbSet<RevokedToken> RevokedTokens => Set<RevokedToken>();

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        modelBuilder.Entity<Customer>()
            .HasIndex(c => c.Email)
            .IsUnique()
            .HasDatabaseName("idx_customers_email");

        modelBuilder.Entity<RevokedToken>()
            .HasIndex(t => t.ExpiresAt)
            .HasDatabaseName("idx_revoked_tokens_expires_at");
    }
}
