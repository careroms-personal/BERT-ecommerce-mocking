using ApiPayment.Models;

namespace ApiPayment.Services;

public class MockPaymentProvider
{
    private readonly double _successRate;
    private readonly double _declineRate;
    private readonly int _timeoutMs;
    private readonly Random _rng = new();

    public MockPaymentProvider(IConfiguration config)
    {
        _successRate = double.Parse(config["Payment:SuccessRate"] ?? "0.75");
        _declineRate = double.Parse(config["Payment:DeclineRate"] ?? "0.15");
        _timeoutMs   = int.Parse(config["Payment:ProviderTimeoutMs"] ?? "4000");
    }

    // timeout scenario rate = 1 - successRate - declineRate
    public async Task<ProviderResult> ChargeAsync(decimal amount, CancellationToken ct)
    {
        var roll = _rng.NextDouble();

        if (roll < _successRate)
        {
            // Happy path: simulate network latency 80–400ms
            await Task.Delay(_rng.Next(80, 400), ct);
            return new ProviderResult(true, $"ch_{Guid.NewGuid():N}", null);
        }

        if (roll < _successRate + _declineRate)
        {
            // Decline: 200–600ms
            await Task.Delay(_rng.Next(200, 600), ct);
            var reasons = new[] { "card_declined", "insufficient_funds", "do_not_honor", "expired_card" };
            return new ProviderResult(false, null, reasons[_rng.Next(reasons.Length)]);
        }

        // Timeout: provider hangs beyond our timeout window
        try
        {
            await Task.Delay(_timeoutMs + _rng.Next(500, 2000), ct);
        }
        catch (OperationCanceledException)
        {
            return new ProviderResult(false, null, "provider_timeout", TimedOut: true);
        }
        return new ProviderResult(false, null, "provider_timeout", TimedOut: true);
    }
}
