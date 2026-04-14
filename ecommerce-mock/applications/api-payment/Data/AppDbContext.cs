using ApiPayment.Models;
using Microsoft.EntityFrameworkCore;

namespace ApiPayment.Data;

public class AppDbContext(DbContextOptions<AppDbContext> options) : DbContext(options)
{
    public DbSet<Payment> Payments => Set<Payment>();

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        modelBuilder.Entity<Payment>()
            .HasIndex(p => p.OrderId)
            .HasDatabaseName("idx_payments_order_id");

        modelBuilder.Entity<Payment>()
            .HasIndex(p => p.CustomerId)
            .HasDatabaseName("idx_payments_customer_id");

        modelBuilder.Entity<Payment>()
            .HasIndex(p => p.Status)
            .HasDatabaseName("idx_payments_status");

        modelBuilder.Entity<Payment>()
            .Property(p => p.Amount)
            .HasPrecision(10, 2);
    }
}
