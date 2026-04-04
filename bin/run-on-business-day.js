#!/usr/bin/env node

const { execFileSync } = require('child_process');
const path = require('path');
const fs = require('fs');

const platform = process.platform;
const arch = process.arch;

// Map platform/arch to binary name
let binaryName;
if (platform === 'linux' && arch === 'x64') {
  binaryName = 'run-on-business-day';
} else if (platform === 'linux' && arch === 'arm64') {
  binaryName = 'run-on-business-day-arm64';
} else if (platform === 'win32' && arch === 'x64') {
  binaryName = 'run-on-business-day.exe';
} else {
  console.error(`Unsupported platform: ${platform}-${arch}`);
  console.error('Supported platforms: linux-x64, linux-arm64, win32-x64');
  process.exit(1);
}

const binaryPath = path.join(__dirname, binaryName);

if (!fs.existsSync(binaryPath)) {
  console.error(`Binary not found: ${binaryPath}`);
  console.error('\nTry reinstalling:');
  console.error('  npm install -g usaagi/run-on-business-day');
  process.exit(1);
}

try {
  execFileSync(binaryPath, process.argv.slice(2), {
    stdio: 'inherit',
    windowsHide: false,
  });
} catch (error) {
  // Propagate exit code from binary
  process.exitCode = error.status || 1;
}
