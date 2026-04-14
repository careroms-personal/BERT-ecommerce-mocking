using System.ComponentModel.DataAnnotations;

namespace ApiCustomer.Models;

public record LoginRequest(
    [Required][EmailAddress] string Email,
    [Required] string Password
);

public record RegisterRequest(
    [Required][EmailAddress] string Email,
    [Required][MinLength(8)] string Password,
    [Required] string FirstName,
    [Required] string LastName
);

public record UpdateCustomerRequest(
    string? FirstName,
    string? LastName,
    string? Email
);

public record TokenResponse(
    string AccessToken,
    string RefreshToken,
    DateTime ExpiresAt,
    CustomerDto Customer
);

public record CustomerDto(
    Guid Id,
    string Email,
    string FirstName,
    string LastName,
    DateTime CreatedAt
);

public static class CustomerMapper
{
    public static CustomerDto ToDto(Customer c) => new(
        c.Id, c.Email, c.FirstName, c.LastName, c.CreatedAt
    );
}
