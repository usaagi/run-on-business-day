#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');
const { exec } = require('child_process');
const { promisify } = require('util');

const execAsync = promisify(exec);

async function downloadFile(url, destPath) {
  return new Promise((resolve, reject) => {
    https.get(url, (response) => {
      // Handle redirects
      if (response.statusCode === 301 || response.statusCode === 302) {
        downloadFile(response.headers.location, destPath).then(resolve).catch(reject);
        return;
      }

      if (response.statusCode !== 200) {
        reject(new Error(`HTTP ${response.statusCode}: ${url}`));
        return;
      }

      const file = fs.createWriteStream(destPath);
      response.pipe(file);

      file.on('finish', () => {
        file.close();
        resolve();
      });

      file.on('error', (err) => {
        fs.unlink(destPath, () => {});
        reject(err);
      });
    }).on('error', reject);
  });
}

async function getRelease(owner, repo, version) {
  return new Promise((resolve, reject) => {
    // Use version if specified, otherwise use latest
    const endpoint = version ? `v${version}` : 'latest';
    const path = `/repos/${owner}/${repo}/releases/${endpoint === 'latest' ? 'latest' : `tags/${endpoint}`}`;

    const options = {
      hostname: 'api.github.com',
      path,
      method: 'GET',
      headers: {
        'User-Agent': 'run-on-business-day-installer'
      }
    };

    https.get(options, (response) => {
      let data = '';

      response.on('data', (chunk) => {
        data += chunk;
      });

      response.on('end', () => {
        if (response.statusCode !== 200) {
          reject(new Error(`GitHub API error: ${response.statusCode} for release ${endpoint}`));
          return;
        }

        try {
          const release = JSON.parse(data);
          resolve(release);
        } catch (err) {
          reject(err);
        }
      });
    }).on('error', reject);
  });
}

async function install() {
  const platform = process.platform;
  const arch = process.arch;
  const version = process.env.npm_package_version;

  let assetName;
  if (platform === 'linux' && arch === 'x64') {
    assetName = 'run-on-business-day';
  } else if (platform === 'linux' && arch === 'arm64') {
    assetName = 'run-on-business-day-arm64';
  } else if (platform === 'win32' && arch === 'x64') {
    assetName = 'run-on-business-day.exe';
  } else {
    console.error(`Unsupported platform: ${platform}-${arch}`);
    console.error('Supported: linux-x64, linux-arm64, win32-x64');
    process.exit(1);
  }

  try {
    console.log(`\nDownloading run-on-business-day ${version} for ${platform}-${arch}...`);

    // Get release info (specific version or latest)
    const release = await getRelease('usaagi', 'run-on-business-day', version);
    const asset = release.assets.find((a) => a.name === assetName);

    if (!asset) {
      throw new Error(`Asset not found: ${assetName}\nAvailable assets: ${release.assets.map((a) => a.name).join(', ')}`);
    }

    // Create bin directory if it doesn't exist
    const binDir = path.join(__dirname, '..', 'bin');
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    const binaryPath = path.join(binDir, assetName);

    // Download binary
    await downloadFile(asset.browser_download_url, binaryPath);
    console.log(`✓ Downloaded to ${binaryPath}`);

    // Make executable on Unix-like systems
    if (platform !== 'win32') {
      fs.chmodSync(binaryPath, 0o755);
      console.log('✓ Set executable permissions');
    }

    console.log('\n✓ Installation complete!');
    console.log(`Run: run-on-business-day --help`);
  } catch (error) {
    console.error('\n✗ Installation failed:', error.message);
    process.exit(1);
  }
}

install();
