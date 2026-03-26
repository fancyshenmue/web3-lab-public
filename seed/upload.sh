#!/bin/bash
# Upload seed images & metadata to MinIO
# Requires: make port-forward-minio (localhost:9000)
set -e

echo "Setting up MinIO alias..."
mc alias set web3lab http://localhost:9000 minioadmin minioadmin >/dev/null 2>&1 || true

echo "Creating bucket (if not exists)..."
mc mb --ignore-existing web3lab/web3lab-assets >/dev/null 2>&1

echo "Uploading ERC20 images..."
mc cp --quiet -r seed/images/erc20/ web3lab/web3lab-assets/erc20/

echo "Uploading ERC721 images..."
mc cp --quiet -r seed/images/erc721/ web3lab/web3lab-assets/erc721/images/

echo "Uploading ERC721 metadata (with JSON content-type)..."
for f in seed/metadata/erc721/*; do
  fname=$(basename "$f")
  mc cp --quiet --attr "Content-Type=application/json" "$f" "web3lab/web3lab-assets/erc721/metadata/$fname"
done

echo "Uploading ERC1155 images..."
mc cp --quiet -r seed/images/erc1155/ web3lab/web3lab-assets/erc1155/images/

echo "Uploading ERC1155 metadata (with JSON content-type)..."
for f in seed/metadata/erc1155/*.json; do
  fname=$(basename "$f")
  mc cp --quiet --attr "Content-Type=application/json" "$f" "web3lab/web3lab-assets/erc1155/metadata/$fname"
done

echo "Setting anonymous read policy..."
mc anonymous set download web3lab/web3lab-assets >/dev/null 2>&1

echo "✅ Upload complete!"
