# Saleae Logic2 Automation Proto (vendored)

This directory vendors `saleae.proto` from the upstream repository `saleae/logic2-automation`.

## Why vendored?

- The Saleae Automation API is documented as **beta** and may change.
- We want a reproducible build and a clear update path (pinning).

## Upstream pin

- Upstream repo: `https://github.com/saleae/logic2-automation`
- File: `proto/saleae/grpc/saleae.proto`
- Pinned commit: `0d7ca19dcc667ca8420ec748d98cf86d4c1f8b78`

## Updating

1. Pick an upstream commit SHA.
2. Replace `saleae.proto` with the upstream version at that SHA.
3. Regenerate Go bindings (see ticket docs) and commit the generated output.


