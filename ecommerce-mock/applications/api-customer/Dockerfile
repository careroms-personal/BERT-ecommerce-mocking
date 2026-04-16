FROM mcr.microsoft.com/dotnet/sdk:9.0-alpine AS builder

WORKDIR /app

# Restore deps first (layer cache)
COPY *.csproj ./
RUN dotnet restore

COPY . .
RUN dotnet publish -c Release -o /publish --no-restore

# ─────────────────────────────────────────────
FROM mcr.microsoft.com/dotnet/aspnet:9.0-alpine

RUN apk add --no-cache curl tzdata icu-libs

WORKDIR /app
COPY --from=builder /publish .

ENV ASPNETCORE_URLS=http://+:8082
ENV DOTNET_SYSTEM_GLOBALIZATION_INVARIANT=false

EXPOSE 8082

ENTRYPOINT ["dotnet", "api-customer.dll"]
