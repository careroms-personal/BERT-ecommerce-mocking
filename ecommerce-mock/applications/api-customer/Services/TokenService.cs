using System.IdentityModel.Tokens.Jwt;
using System.Security.Claims;
using System.Text;
using ApiCustomer.Models;
using Microsoft.IdentityModel.Tokens;

namespace ApiCustomer.Services;

public class TokenService
{
    private readonly string _secret;
    private readonly int _accessExpiryMinutes;
    private readonly int _refreshExpiryDays;
    private readonly SymmetricSecurityKey _key;

    public TokenService(IConfiguration config)
    {
        _secret = config["Jwt:Secret"] ?? throw new InvalidOperationException("Jwt:Secret not configured");
        _accessExpiryMinutes = int.Parse(config["Jwt:AccessTokenExpiryMinutes"] ?? "60");
        _refreshExpiryDays = int.Parse(config["Jwt:RefreshTokenExpiryDays"] ?? "7");
        _key = new SymmetricSecurityKey(Encoding.UTF8.GetBytes(_secret));
    }

    public (string token, Guid jti, DateTime expiresAt) GenerateAccessToken(Customer customer)
    {
        var jti = Guid.NewGuid();
        var expiresAt = DateTime.UtcNow.AddMinutes(_accessExpiryMinutes);

        var claims = new[]
        {
            new Claim(JwtRegisteredClaimNames.Sub, customer.Id.ToString()),
            new Claim(JwtRegisteredClaimNames.Email, customer.Email),
            new Claim(JwtRegisteredClaimNames.Jti, jti.ToString()),
            new Claim("first_name", customer.FirstName),
        };

        var token = new JwtSecurityToken(
            claims: claims,
            expires: expiresAt,
            signingCredentials: new SigningCredentials(_key, SecurityAlgorithms.HmacSha256)
        );

        return (new JwtSecurityTokenHandler().WriteToken(token), jti, expiresAt);
    }

    public (string token, Guid jti, DateTime expiresAt) GenerateRefreshToken(Customer customer)
    {
        var jti = Guid.NewGuid();
        var expiresAt = DateTime.UtcNow.AddDays(_refreshExpiryDays);

        var claims = new[]
        {
            new Claim(JwtRegisteredClaimNames.Sub, customer.Id.ToString()),
            new Claim(JwtRegisteredClaimNames.Jti, jti.ToString()),
            new Claim("token_type", "refresh"),
        };

        var token = new JwtSecurityToken(
            claims: claims,
            expires: expiresAt,
            signingCredentials: new SigningCredentials(_key, SecurityAlgorithms.HmacSha256)
        );

        return (new JwtSecurityTokenHandler().WriteToken(token), jti, expiresAt);
    }

    public ClaimsPrincipal? ValidateToken(string token, out Guid jti, out bool expired)
    {
        jti = Guid.Empty;
        expired = false;

        try
        {
            var handler = new JwtSecurityTokenHandler();
            var principal = handler.ValidateToken(token, new TokenValidationParameters
            {
                ValidateIssuer = false,
                ValidateAudience = false,
                ValidateLifetime = true,
                ValidateIssuerSigningKey = true,
                IssuerSigningKey = _key,
                ClockSkew = TimeSpan.Zero
            }, out var validatedToken);

            var jtiClaim = principal.FindFirst(JwtRegisteredClaimNames.Jti)?.Value;
            Guid.TryParse(jtiClaim, out jti);
            return principal;
        }
        catch (SecurityTokenExpiredException)
        {
            expired = true;
            return null;
        }
        catch
        {
            return null;
        }
    }
}
