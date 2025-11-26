#!/usr/bin/env python3
"""
Production Codebase Hash Generator
Generates rock-solid SHA256 hash of all relevant codebase components
Excludes non-code files, build artifacts, and temporary files
"""

import os
import sys
import hashlib
import json
from pathlib import Path
from typing import Dict, List, Set

# Directories and files to include in hash calculation
INCLUDE_DIRS = {
    'cmd',
    'pkg', 
    'internal',
    'scripts',
    'docs',
    'config',
    'web',
    'static',
    'templates',
    'go.mod',
    'go.sum',
    'Makefile',
    'Dockerfile',
    'docker-compose.yml',
    '.github',
    'requirements.txt',
    'package.json',
}

# File patterns to exclude from hash calculation
EXCLUDE_PATTERNS = {
    '*.log',
    '*.tmp',
    '*.bak',
    '*.swp',
    '*.swo',
    '*~',
    '.DS_Store',
    'Thumbs.db',
    '*.pid',
    '*.lock',
    '.#*',
    '#*#',
    '*.orig',
    '*.rej',
    '.git',
    '.svn',
    'node_modules',
    '__pycache__',
    '.pytest_cache',
    '.coverage',
    'coverage.xml',
    '*.pyc',
    '*.pyo',
    '*.pyd',
    '.env',
    '.env.*',
    'dist',
    'build',
    'target',
    'vendor',
    '.terraform',
    '*.tfstate',
    '*.tfstate.*',
    '.tox',
    '.venv',
    'venv',
    'env',
    '.idea',
    '.vscode',
    '*.sublime-project',
    '*.sublime-workspace',
}

# Include these files even if they match exclude patterns
FORCE_INCLUDE = {
    '.gitignore',
    '.gitattributes',
    'go.mod',
    'go.sum',
}

def should_include_file(file_path: Path, base_dir: Path) -> bool:
    """Check if file should be included in hash calculation"""
    
    # Get relative path from base directory
    rel_path = file_path.relative_to(base_dir)
    rel_str = str(rel_path)
    
    # Check if file is in force include list
    if file_path.name in FORCE_INCLUDE:
        return True
    
    # Check if file matches any exclude pattern
    for pattern in EXCLUDE_PATTERNS:
        if file_path.match(pattern):
            return False
        if rel_path.match(pattern):
            return False
    
    # Check if file is in included directories or is a root file
    parts = rel_path.parts
    if not parts:  # Root directory
        return False
        
    first_part = parts[0]
    if first_part.startswith('.'):  # Hidden directories/files
        return False
        
    # Check if first part is in include directories or if it's a root file
    if first_part in INCLUDE_DIRS or len(parts) == 1:
        return True
        
    # Check if any parent directory is in include list
    for part in parts:
        if part in INCLUDE_DIRS:
            return True
            
    return False

def calculate_file_hash(file_path: Path) -> str:
    """Calculate SHA256 hash of a single file"""
    hash_sha256 = hashlib.sha256()
    
    try:
        with open(file_path, 'rb') as f:
            # Read file in chunks to handle large files
            for chunk in iter(lambda: f.read(4096), b""):
                hash_sha256.update(chunk)
        return hash_sha256.hexdigest()
    except (OSError, IOError) as e:
        print(f"Warning: Could not read {file_path}: {e}", file=sys.stderr)
        return None

def calculate_codebase_hash(base_dir: str = '.') -> str:
    """Calculate comprehensive hash of the codebase"""
    
    base_path = Path(base_dir).resolve()
    print(f"Calculating codebase hash for: {base_path}", file=sys.stderr)
    
    # Collect all files to hash
    files_to_hash = []
    
    # Find all relevant files
    for root, dirs, files in os.walk(base_path):
        root_path = Path(root)
        
        # Skip hidden directories
        dirs[:] = [d for d in dirs if not d.startswith('.')]
        
        for file_name in files:
            file_path = root_path / file_name
            
            if should_include_file(file_path, base_path):
                files_to_hash.append(file_path)
    
    # Sort files for consistent ordering
    files_to_hash.sort()
    
    print(f"Found {len(files_to_hash)} files to hash", file=sys.stderr)
    
    # Calculate individual file hashes
    file_hashes = {}
    total_hash = hashlib.sha256()
    
    for file_path in files_to_hash:
        rel_path = str(file_path.relative_to(base_path))
        file_hash = calculate_file_hash(file_path)
        
        if file_hash:
            file_hashes[rel_path] = file_hash
            
            # Add to total hash with path for uniqueness
            hash_input = f"{rel_path}:{file_hash}"
            total_hash.update(hash_input.encode('utf-8'))
    
    # Generate final hash
    final_hash = total_hash.hexdigest()
    
    # Also create detailed hash report
    hash_report = {
        'base_directory': str(base_path),
        'total_hash': final_hash,
        'algorithm': 'SHA256',
        'files_count': len(file_hashes),
        'timestamp': str(Path().resolve()),
        'file_hashes': file_hashes
    }
    
    # Save detailed report
    report_path = base_path / 'codebase_hash_report.json'
    with open(report_path, 'w') as f:
        json.dump(hash_report, f, indent=2, sort_keys=True)
    
    print(f"Codebase hash: {final_hash[:16]}...", file=sys.stderr)
    print(f"Hash report saved to: {report_path}", file=sys.stderr)
    
    return final_hash

def verify_codebase_consistency(base_dir: str = '.') -> Dict:
    """Verify codebase consistency and return detailed report"""
    
    base_path = Path(base_dir).resolve()
    
    # Calculate current hash
    current_hash = calculate_codebase_hash(base_dir)
    
    # Load previous hash if available
    hash_file = base_path / '.codebase_hash'
    previous_hash = None
    
    if hash_file.exists():
        with open(hash_file, 'r') as f:
            previous_hash = f.read().strip()
    
    # Check for uncommitted changes if git repository
    git_status = None
    if (base_path / '.git').exists():
        try:
            import subprocess
            result = subprocess.run(
                ['git', 'status', '--porcelain'],
                cwd=base_path,
                capture_output=True,
                text=True
            )
            git_status = result.stdout.strip()
        except:
            pass
    
    report = {
        'current_hash': current_hash,
        'previous_hash': previous_hash,
        'hash_match': current_hash == previous_hash if previous_hash else None,
        'git_status': git_status,
        'timestamp': str(Path().resolve()),
        'base_directory': str(base_path)
    }
    
    return report

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 codebase_hasher.py <command> [directory]")
        print("Commands:")
        print("  calculate    Calculate codebase hash")
        print("  verify       Verify codebase consistency")
        print("  update       Save current hash as baseline")
        sys.exit(1)
    
    command = sys.argv[1]
    directory = sys.argv[2] if len(sys.argv) > 2 else '.'
    
    if command == 'calculate':
        hash_value = calculate_codebase_hash(directory)
        print(hash_value)
        
    elif command == 'verify':
        report = verify_codebase_consistency(directory)
        print(json.dumps(report, indent=2))
        
        if report['hash_match'] is False:
            print(f"\nERROR: Codebase has changed!", file=sys.stderr)
            print(f"Previous: {report['previous_hash'][:16]}...", file=sys.stderr)
            print(f"Current:  {report['current_hash'][:16]}...", file=sys.stderr)
            sys.exit(1)
            
    elif command == 'update':
        hash_value = calculate_codebase_hash(directory)
        hash_file = Path(directory) / '.codebase_hash'
        with open(hash_file, 'w') as f:
            f.write(hash_value)
        print(f"Hash baseline updated: {hash_value[:16]}...")
        
    else:
        print(f"Unknown command: {command}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()