# Golingko Database Migration Runner
param (
    [Parameter(Position = 0)]
    [string]$Action = "up",
    [Parameter(Position = 1, ValueFromRemainingArguments = $true)]
    [string[]]$ExtraArgs
)

& go run ./cmd/migrate $Action @ExtraArgs
exit $LASTEXITCODE
