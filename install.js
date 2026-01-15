#!/usr/bin/env node

/**
 * SimpleArchiver npm installer
 * Downloads the pre-compiled binary for the current platform
 */

const https = require('https');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const REPO = 'Lynricsy/SimpleArchiver';
const BINARY_NAME = 'simple-archiver';

// Map Node.js platform/arch to Go platform/arch
function getPlatformInfo() {
  const platform = process.platform;
  const arch = process.arch;

  const platformMap = {
    darwin: 'darwin',
    linux: 'linux',
    win32: 'windows',
  };

  const archMap = {
    x64: 'amd64',
    arm64: 'arm64',
    ia32: '386',
  };

  const goPlatform = platformMap[platform];
  const goArch = archMap[arch];

  if (!goPlatform || !goArch) {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }

  return {
    platform: goPlatform,
    arch: goArch,
    ext: platform === 'win32' ? '.exe' : '',
  };
}

// Get the latest release version from GitHub
async function getLatestVersion() {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: 'api.github.com',
      path: `/repos/${REPO}/releases/latest`,
      headers: {
        'User-Agent': 'simple-archiver-installer',
      },
    };

    https.get(options, (res) => {
      let data = '';
      res.on('data', (chunk) => (data += chunk));
      res.on('end', () => {
        try {
          const json = JSON.parse(data);
          resolve(json.tag_name);
        } catch (e) {
          reject(new Error('Failed to parse release info'));
        }
      });
    }).on('error', reject);
  });
}

// Download file from URL
async function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);

    const request = (url) => {
      https.get(url, (res) => {
        // Handle redirects
        if (res.statusCode === 302 || res.statusCode === 301) {
          request(res.headers.location);
          return;
        }

        if (res.statusCode !== 200) {
          reject(new Error(`Download failed with status ${res.statusCode}`));
          return;
        }

        res.pipe(file);
        file.on('finish', () => {
          file.close();
          resolve();
        });
      }).on('error', (err) => {
        fs.unlink(dest, () => {});
        reject(err);
      });
    };

    request(url);
  });
}

async function main() {
  try {
    console.log('📦 Installing SimpleArchiver...');

    const { platform, arch, ext } = getPlatformInfo();
    console.log(`   Platform: ${platform}-${arch}`);

    // Get latest version
    const version = await getLatestVersion();
    console.log(`   Version: ${version}`);

    // Create bin directory
    const binDir = path.join(__dirname, 'bin');
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    // Download binary
    const binaryName = `${BINARY_NAME}_${platform}_${arch}${ext}`;
    const downloadUrl = `https://github.com/${REPO}/releases/download/${version}/${binaryName}`;
    const destPath = path.join(binDir, BINARY_NAME + ext);

    console.log(`   Downloading from: ${downloadUrl}`);
    await downloadFile(downloadUrl, destPath);

    // Make executable on Unix
    if (process.platform !== 'win32') {
      fs.chmodSync(destPath, 0o755);
    }

    console.log('✅ SimpleArchiver installed successfully!');
    console.log(`   Run with: npx simple-archiver`);
  } catch (err) {
    console.error('❌ Installation failed:', err.message);
    console.error('   You can manually download from:');
    console.error(`   https://github.com/${REPO}/releases`);
    process.exit(1);
  }
}

main();
