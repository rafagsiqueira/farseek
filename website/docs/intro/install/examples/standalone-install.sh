# Download the installer script:
curl --proto '=https' --tlsv1.2 -fsSL https://get.farseek.dev/install-farseek.sh -o install-farseek.sh
# Alternatively: wget --secure-protocol=TLSv1_2 --https-only https://get.farseek.dev/install-farseek.sh -O install-farseek.sh

# Grant execution permissions:
chmod +x install-farseek.sh

# Please inspect the downloaded script at this point.

# Run the installer:
./install-farseek.sh --install-method standalone

# Remove the installer:
rm -f install-farseek.sh