#!/usr/bin/env node
const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');

// Use Docker instead of local binary
// Get current working directory in a cross-platform way
const currentDir = process.cwd().replace(/\\/g, '/');
const goArchLintCmd = `docker run --rm -v "${currentDir}:/app" fe3dback/go-arch-lint:latest-stable-release`;

// Check if Docker is available
try {
    execSync('docker --version', { 
        stdio: 'ignore',
        shell: true 
    });
} catch {
    console.error('Docker not found. Please install Docker to run architecture linting.');
    console.error('Alternatively, install go-arch-lint locally: go install github.com/fe3dback/go-arch-lint@latest');
    process.exit(1);
}

// Check if config exists
if (!fs.existsSync('.go-arch-lint.yml')) {
    console.error('.go-arch-lint.yml not found in project root');
    process.exit(1);
}

try {
    // Run go-arch-lint via Docker and capture output
    const result = execSync(`${goArchLintCmd} check --project-path /app --json`, { 
        stdio: ['inherit', 'pipe', 'pipe'],
        encoding: 'utf8',
        timeout: 300000,
        shell: true
    });
    
    // Parse the JSON result
    try {
        const parsedResult = JSON.parse(result);
        const warnings = parsedResult.Payload?.ArchWarningsDeps || [];
        
        if (warnings.length === 0) {
            // No violations found
            if (process.stdout.isTTY) {
                console.log('✅ Architecture validation passed');
            }
            process.exit(0);
        } else {
            // Violations found - this shouldn't happen with exit code 0, but handle it
            if (process.stdout.isTTY) {
                console.error(`❌ Found ${warnings.length} architecture violation(s):`);
            }
            
            // Convert to VS Code problem format
            warnings.forEach(warning => {
                const file = warning.FileAbsolutePath || warning.FileRelativePath;
                const line = warning.Reference?.Line || 1;
                const column = warning.Reference?.Offset || 1;
                
                // Convert to relative path and normalize separators for cross-platform
                const relativePath = path.relative(process.cwd(), file).replace(/\\/g, '/');
                
                // More descriptive error message
                const component = warning.ComponentName;
                const importPath = warning.ResolvedImportName;
                const message = `Architecture violation: '${component}' component cannot import '${importPath}'`;
                
                // VS Code problem format: file:line:column: severity: message
                console.log(`${relativePath}:${line}:${column}: error: ${message}`);
            });
            
            process.exit(1);
        }
    } catch (parseError) {
        if (process.stdout.isTTY) {
            console.error('Failed to parse go-arch-lint JSON output:');
            console.error(result);
        }
        process.exit(1);
    }
    
} catch (error) {
    // This happens when go-arch-lint exits with non-zero code (usually when violations found)
    const output = error.stdout || error.stderr || '';
    
    try {
        const result = JSON.parse(output);
        const warnings = result.Payload?.ArchWarningsDeps || [];
        
        if (warnings.length === 0) {
            if (process.stdout.isTTY) {
                console.log('✅ Architecture validation passed');
            }
            process.exit(0);
        }
        
        // Only show count in TTY mode (manual runs), not on save
        if (process.stdout.isTTY) {
            console.error(`❌ Found ${warnings.length} architecture violation(s):`);
        }
        
        // Convert to VS Code problem format
        warnings.forEach(warning => {
            const file = warning.FileAbsolutePath || warning.FileRelativePath;
            const line = warning.Reference?.Line || 1;
            const column = warning.Reference?.Offset || 1;
            
            // Convert to relative path and normalize separators for cross-platform
            const relativePath = path.relative(process.cwd(), file).replace(/\\/g, '/');
            
            // More descriptive error message
            const component = warning.ComponentName;
            const importPath = warning.ResolvedImportName;
            const message = `Architecture violation: '${component}' component cannot import '${importPath}'`;
            
            // VS Code problem format: file:line:column: severity: message
            console.log(`${relativePath}:${line}:${column}: error: ${message}`);
        });
        
        process.exit(1);
        
    } catch (parseError) {
        if (process.stdout.isTTY) {
            console.error('Failed to parse go-arch-lint JSON output:');
            console.error(output);
        }
        process.exit(1);
    }
}
