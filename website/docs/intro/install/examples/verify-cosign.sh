FARSEEEK_VERSION_MAJORMINOR="Add your Farseek major and minor version here"
IDENTITY="https://github.com/rafagsiqueira/farseek/.github/workflows/release.yml@refs/heads/v${FARSEEEK_VERSION_MAJORMINOR}"
# For alpha and beta builds use /main
cosign \
    verify-blob \
    --certificate-identity "${IDENTITY}" \
    --signature farseek_*.sig \
    --certificate farseek_*.pem \
    --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
    farseek_*_SHA256SUMS
