using ApiCustomer.Data;
using ApiCustomer.Models;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;
using Serilog.Context;

namespace ApiCustomer.Controllers;

[ApiController]
[Route("customers")]
public class CustomerController(AppDbContext db, ILogger<CustomerController> logger) : ControllerBase
{
    [HttpPost("register")]
    public async Task<IActionResult> Register([FromBody] RegisterRequest req)
    {
        var requestId = HttpContext.TraceIdentifier;

        using (LogContext.PushProperty("Category", "AUTH"))
        using (LogContext.PushProperty("RequestId", requestId))
        using (LogContext.PushProperty("Email", req.Email))
        {
            logger.LogInformation("Registration attempt for {Email}", req.Email);

            var exists = await db.Customers.AnyAsync(c => c.Email == req.Email);
            if (exists)
            {
                using (LogContext.PushProperty("Category", "ERROR"))
                {
                    logger.LogWarning("Registration failed — email already exists {Email}", req.Email);
                }
                return Conflict(new { error = "email already registered" });
            }

            var customer = new Customer
            {
                Email = req.Email,
                PasswordHash = BCrypt.Net.BCrypt.HashPassword(req.Password),
                FirstName = req.FirstName,
                LastName = req.LastName,
            };

            db.Customers.Add(customer);

            try
            {
                await db.SaveChangesAsync();
            }
            catch (Exception ex)
            {
                using (LogContext.PushProperty("Category", "DB_ERROR"))
                {
                    logger.LogError(ex, "Failed to register customer {Email}", req.Email);
                }
                return StatusCode(500, new { error = "database error" });
            }

            using (LogContext.PushProperty("CustomerId", customer.Id))
            {
                logger.LogInformation("Customer registered {Email}", req.Email);
            }

            return Created($"/customers/{customer.Id}", CustomerMapper.ToDto(customer));
        }
    }

    [HttpGet("{id:guid}")]
    public async Task<IActionResult> GetCustomer(Guid id)
    {
        var requestId = HttpContext.TraceIdentifier;

        using (LogContext.PushProperty("Category", "AUTH"))
        using (LogContext.PushProperty("RequestId", requestId))
        using (LogContext.PushProperty("CustomerId", id))
        {
            logger.LogInformation("Fetching customer profile {CustomerId}", id);

            var customer = await db.Customers.FindAsync(id);
            if (customer is null)
            {
                using (LogContext.PushProperty("Category", "NOT_FOUND"))
                {
                    logger.LogWarning("Customer not found {CustomerId}", id);
                }
                return NotFound(new { error = "customer not found", customer_id = id });
            }

            logger.LogInformation("Customer profile fetched {CustomerId} {Email}", id, customer.Email);
            return Ok(CustomerMapper.ToDto(customer));
        }
    }

    [HttpPut("{id:guid}")]
    public async Task<IActionResult> UpdateCustomer(Guid id, [FromBody] UpdateCustomerRequest req)
    {
        var requestId = HttpContext.TraceIdentifier;

        using (LogContext.PushProperty("Category", "AUTH"))
        using (LogContext.PushProperty("RequestId", requestId))
        using (LogContext.PushProperty("CustomerId", id))
        {
            logger.LogInformation("Updating customer profile {CustomerId}", id);

            var customer = await db.Customers.FindAsync(id);
            if (customer is null)
            {
                using (LogContext.PushProperty("Category", "NOT_FOUND"))
                {
                    logger.LogWarning("Customer not found for update {CustomerId}", id);
                }
                return NotFound(new { error = "customer not found", customer_id = id });
            }

            if (req.Email is not null && req.Email != customer.Email)
            {
                var emailTaken = await db.Customers.AnyAsync(c => c.Email == req.Email && c.Id != id);
                if (emailTaken)
                {
                    using (LogContext.PushProperty("Category", "ERROR"))
                    {
                        logger.LogWarning("Update failed — email already taken {Email}", req.Email);
                    }
                    return Conflict(new { error = "email already in use" });
                }
                customer.Email = req.Email;
            }

            if (req.FirstName is not null) customer.FirstName = req.FirstName;
            if (req.LastName is not null) customer.LastName = req.LastName;
            customer.UpdatedAt = DateTime.UtcNow;

            try
            {
                await db.SaveChangesAsync();
            }
            catch (Exception ex)
            {
                using (LogContext.PushProperty("Category", "DB_ERROR"))
                {
                    logger.LogError(ex, "Failed to update customer {CustomerId}", id);
                }
                return StatusCode(500, new { error = "database error" });
            }

            logger.LogInformation("Customer profile updated {CustomerId}", id);
            return Ok(CustomerMapper.ToDto(customer));
        }
    }
}
