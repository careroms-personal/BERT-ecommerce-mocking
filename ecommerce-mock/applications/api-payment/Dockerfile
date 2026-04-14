FROM mcr.microsoft.com/dotnet/sdk:9.0-alpine AS builder

WORKDIR /app

COPY *.csproj ./
RUN dotnet restore

COPY . .
RUN dotnet publish -c Release -o /publish --no-restore

# ─────────────────────────────────────────────
FROM mcr.microsoft.com/dotnet/aspnet:9.0-alpine

RUN apk add --no-cache curl tzdata

WORKDIR /app
COPY --from=builder /publish .

ENV ASPNETCORE_URLS=http://+:8085
ENV DOTNET_SYSTEM_GLOBALIZATION_INVARIANT=false

EXPOSE 8085

HEALTHCHECK --interval=15s --timeout=5s --start-period=30s --retries=3 \
    CMD curl -sf http://localhost:8085/health || exit 1

ENTRYPOINT ["dotnet", "api-payment.dll"]
