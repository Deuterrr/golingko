# Golingko Transit Database Seeder
param (
    [Parameter(Position = 0)]
    [ValidateSet("up", "down", "reset")]
    [string]$Action = "up"
)

$flag = switch ($Action) {
    "down"  { "-down" }
    "reset" { "-reset" }
    default { "" }
}

if ($flag) {
    & go run ./cmd/seed $flag
} else {
    & go run ./cmd/seed
}
exit $LASTEXITCODE
