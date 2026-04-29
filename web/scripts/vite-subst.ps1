param(
  [Parameter(Mandatory = $true)]
  [ValidateSet('dev', 'build', 'preview')]
  [string]$Mode,

  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]]$ExtraArgs
)

$ErrorActionPreference = 'Stop'

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptDir
$repoRoot = Split-Path -Parent $projectRoot
$projectLeaf = Split-Path -Leaf $projectRoot

function Test-AsciiPath {
  param([Parameter(Mandatory = $true)][string]$Path)

  return $Path -notmatch '[^\u0000-\u007F]'
}

function Get-FreeDriveLetter {
  foreach ($letter in @('X', 'Y', 'Z', 'W', 'V')) {
    if (-not (Test-Path -LiteralPath "${letter}:\")) {
      return $letter
    }
  }

  throw 'No free drive letter available for subst.'
}

function Find-AsciiAlias {
  param(
    [Parameter(Mandatory = $true)][string]$RepoRoot,
    [Parameter(Mandatory = $true)][string]$ProjectLeaf
  )

  if (Test-AsciiPath $projectRoot) {
    return $projectRoot
  }

  $repoParent = Split-Path -Parent $RepoRoot
  $resolvedRepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path

  foreach ($candidateRoot in (Get-ChildItem -LiteralPath $repoParent -Directory -Force)) {
    if (-not ($candidateRoot.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
      continue
    }

    $target = $candidateRoot.Target
    if ($target -is [array]) {
      $target = $target[0]
    }

    if ([string]::IsNullOrWhiteSpace($target)) {
      continue
    }

    try {
      $resolvedTarget = (Resolve-Path -LiteralPath $target).Path
    }
    catch {
      continue
    }

    if ($resolvedTarget -ne $resolvedRepoRoot) {
      continue
    }

    $candidatePath = Join-Path $candidateRoot.FullName $ProjectLeaf
    if ((Test-Path -LiteralPath $candidatePath) -and (Test-AsciiPath $candidatePath)) {
      return $candidatePath
    }
  }

  return $null
}

$workRoot = $projectRoot
$configRoot = $projectRoot
$cleanupDrive = $null

if (-not (Test-AsciiPath $projectRoot)) {
  $asciiAlias = Find-AsciiAlias -RepoRoot $repoRoot -ProjectLeaf $projectLeaf
  if ($asciiAlias) {
    $configRoot = $asciiAlias
  }
  else {
    try {
      $driveLetter = Get-FreeDriveLetter
      & subst "${driveLetter}:" $projectRoot | Out-Null
      if ($LASTEXITCODE -eq 0 -and (Test-Path -LiteralPath "${driveLetter}:\")) {
        $workRoot = "${driveLetter}:\"
        $configRoot = $workRoot
        $cleanupDrive = "${driveLetter}:"
      }
    }
    catch {
      Write-Warning "Failed to create subst drive. Falling back to direct path: $($_.Exception.Message)"
    }
  }
}

$configPath = Join-Path $configRoot 'vite.config.ts'

try {
  Push-Location -LiteralPath $workRoot

  switch ($Mode) {
    'build' {
      & node .\node_modules\typescript\bin\tsc -b
      if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

      & node .\node_modules\vite\bin\vite.js build --config $configPath @ExtraArgs
      exit $LASTEXITCODE
    }
    'dev' {
      & node .\node_modules\vite\bin\vite.js dev --config $configPath @ExtraArgs
      exit $LASTEXITCODE
    }
    'preview' {
      & node .\node_modules\vite\bin\vite.js preview --config $configPath @ExtraArgs
      exit $LASTEXITCODE
    }
  }
}
finally {
  Pop-Location -ErrorAction SilentlyContinue
  if ($cleanupDrive) {
    & subst $cleanupDrive /D | Out-Null
  }
}
