$version = [version]"YOUR_FARSEEEK_VERSION"
$identity = "https://github.com/rafagsiqueira/farseek/.github/workflows/release.yml@refs/heads/v${version.Major}.${version.Minor}"
# For alpha and beta builds use /main
cosign.exe `
    verify-blob `
    --certificate-identity $identity `
    --signature "farseek_YOURVERSION_REPLACEME.sig" `
    --certificate "farseek_YOURVERSION_REPLACEME.pem" `
    --certificate-oidc-issuer "https://token.actions.githubusercontent.com" `
    "farseek_YOURVERSION_REPLACEME_SHA256SUMS"