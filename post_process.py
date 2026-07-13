#!/usr/bin/env python3
import os
import sys
import shutil
import zipfile
import subprocess

def get_largest_file_info_in_zip(zip_path):
    try:
        with zipfile.ZipFile(zip_path, 'r') as z:
            largest_size = -1
            largest_name = None
            for info in z.infolist():
                if info.is_dir():
                    continue
                if info.file_size > largest_size:
                    largest_size = info.file_size
                    largest_name = info.filename
            if largest_name:
                return largest_name, largest_size
    except Exception as e:
        print(f"Error reading zip {zip_path}: {e}")
    return None, 0

def get_largest_file_info_via_7z(archive_path):
    try:
        # Check if 7z or 7za is available
        cmd_name = '7z'
        if not shutil.which('7z') and shutil.which('7za'):
            cmd_name = '7za'
        
        result = subprocess.run([cmd_name, 'l', archive_path], capture_output=True, text=True, check=True)
        lines = result.stdout.splitlines()
        
        table_lines = []
        in_table = False
        for line in lines:
            if line.startswith('------------------- ----- ------------ ------------'):
                if not in_table:
                    in_table = True
                else:
                    in_table = False
                continue
            if in_table:
                table_lines.append(line)
        
        largest_size = -1
        largest_name = None
        for line in table_lines:
            if len(line) < 54:
                continue
            attr = line[20:25]
            if 'D' in attr: # Directory
                continue
            size_str = line[26:38].strip()
            name_str = line[53:].strip()
            if size_str.isdigit():
                size = int(size_str)
                if size > largest_size:
                    largest_size = size
                    largest_name = name_str
        return largest_name, largest_size
    except Exception as e:
        print(f"Error running 7z/7za l for {archive_path}: {e}")
    return None, 0

def get_largest_file_info_in_dir(dir_path):
    largest_size = -1
    largest_path = None
    for root, _, files in os.walk(dir_path):
        for f in files:
            path = os.path.join(root, f)
            try:
                size = os.path.getsize(path)
                if size > largest_size:
                    largest_size = size
                    largest_path = path
            except Exception:
                continue
    if largest_path:
        return largest_path, largest_size
    return None, 0

def get_platform_for_item(item_path, console_name=None):
    filename = os.path.basename(item_path).lower()
    ext = ""
    size = 0

    if os.path.isfile(item_path):
        if filename.endswith('.zip'):
            inner_name, inner_size = get_largest_file_info_in_zip(item_path)
            if inner_name:
                ext = os.path.splitext(inner_name)[1].lower()
                size = inner_size
            else:
                ext = '.zip'
                size = os.path.getsize(item_path)
        elif filename.endswith(('.rar', '.7z', '.tar.gz', '.tgz', '.gz')):
            inner_name, inner_size = get_largest_file_info_via_7z(item_path)
            if inner_name:
                ext = os.path.splitext(inner_name)[1].lower()
                size = inner_size
            else:
                ext = os.path.splitext(filename)[1].lower()
                size = os.path.getsize(item_path)
        else:
            ext = os.path.splitext(filename)[1].lower()
            size = os.path.getsize(item_path)
    elif os.path.isdir(item_path):
        largest_path, largest_size = get_largest_file_info_in_dir(item_path)
        if largest_path:
            ext = os.path.splitext(largest_path)[1].lower()
            size = largest_size
        else:
            return None
    else:
        return None

    # Platform Mapping Table
    EXTENSION_MAP = {
        '.sfc': 'snes',
        '.smc': 'snes',
        '.n64': 'n64',
        '.z64': 'n64',
        '.nds': 'nds',
        '.nes': 'nes',
        '.gba': 'gba',
        '.iso': 'ps1_or_ps2',
        '.chd': 'ps1_or_ps2',
    }

    platform = EXTENSION_MAP.get(ext)
    if not platform:
        return None

    if platform == 'ps1_or_ps2':
        c_name = (console_name or "").lower()
        f_name = filename.lower()
        if 'playstation 2' in c_name or 'ps2' in c_name or 'playstation 2' in f_name or 'ps2' in f_name:
            return 'ps2'
        elif 'playstation' in c_name or 'psx' in c_name or 'ps1' in c_name or 'playstation' in f_name or 'psx' in f_name or 'ps1' in f_name:
            return 'ps1'
        else:
            # Fallback based on size: PS1 <= 750MB, PS2 > 750MB
            if size > 786432000:
                return 'ps2'
            else:
                return 'ps1'

    return platform

def ensure_dir_and_chown(path):
    if not os.path.exists(path):
        os.makedirs(path, exist_ok=True)
        try:
            os.chown(path, 1000, 1000)
        except Exception as e:
            print(f"Warning: chown for {path} failed: {e}")

def move_and_chown(src, dest):
    shutil.move(src, dest)
    try:
        if os.path.isdir(dest):
            os.chown(dest, 1000, 1000)
            for root, dirs, files in os.walk(dest):
                for d in dirs:
                    os.chown(os.path.join(root, d), 1000, 1000)
                for f in files:
                    os.chown(os.path.join(root, f), 1000, 1000)
        else:
            os.chown(dest, 1000, 1000)
    except Exception as e:
        print(f"Warning: chown for {dest} failed: {e}")

def trigger_retrom_update():
    api_addr = os.environ.get("RETROM_API_ADDR")
    if not api_addr:
        print("RETROM_API_ADDR environment variable is not set. Skipping library update.")
        return

    # Add Go bin to PATH where grpcurl might be installed
    home_dir = os.path.expanduser("~")
    go_bin = os.path.join(home_dir, "go", "bin")
    if os.path.exists(go_bin) and go_bin not in os.environ.get("PATH", ""):
        os.environ["PATH"] = os.environ["PATH"] + os.pathsep + go_bin

    grpcurl_path = shutil.which('grpcurl')
    if not grpcurl_path:
        print("grpcurl is not installed or not in PATH. Skipping library update.")
        return

    print(f"Triggering Retrom library update via gRPC at {api_addr}...")
    try:
        cmd = [grpcurl_path, '-plaintext', '-d', '{}', api_addr, 'retrom.LibraryService/UpdateLibrary']
        result = subprocess.run(cmd, capture_output=True, text=True)
        if result.returncode == 0:
            print("Retrom library update triggered successfully.")
            print(result.stdout)
        else:
            print(f"Failed to trigger Retrom library update: {result.stderr}")
    except Exception as e:
        print(f"Error triggering Retrom library update: {e}")
def main():
    dest_root = os.environ.get("DOWNLOADS_DIR")
    if not dest_root:
        dest_root = "/mnt/games/Roms"
        if not os.path.exists(dest_root):
            dest_root = "./downloads"
    
    dest_root = os.path.abspath(dest_root)
    print(f"Completed download payload directory: {dest_root}")

    # Case 1: Specific path passed as argument
    if len(sys.argv) > 1:
        item_path = sys.argv[1]
        console_name = sys.argv[2] if len(sys.argv) > 2 else None
        
        if not os.path.exists(item_path):
            # Try to resolve relative to dest_root
            alternative_path = os.path.join(dest_root, os.path.basename(item_path))
            if os.path.exists(alternative_path):
                item_path = alternative_path
            else:
                print(f"Error: Path '{item_path}' does not exist.")
                sys.exit(1)

        item_path = os.path.abspath(item_path)
        
        # Don't process if it's already inside a platform subfolder
        parent_dir = os.path.basename(os.path.dirname(item_path))
        platform_dirs = {'snes', 'n64', 'nds', 'nes', 'gba', 'ps1', 'ps2'}
        if parent_dir in platform_dirs:
            print(f"Item '{item_path}' is already inside platform folder '{parent_dir}'.")
            sys.exit(0)

        platform = get_platform_for_item(item_path, console_name)
        if platform:
            target_dir = os.path.join(dest_root, platform)
            ensure_dir_and_chown(target_dir)
            
            dest_path = os.path.join(target_dir, os.path.basename(item_path))
            print(f"Moving '{item_path}' to '{dest_path}'...")
            move_and_chown(item_path, dest_path)
            trigger_retrom_update()
        else:
            print(f"No platform matched for item: '{item_path}'")
    
    # Case 2: Scan the destination directory for loose files
    else:
        print(f"Scanning '{dest_root}' for loose ROMs/games...")
        platform_dirs = {'snes', 'n64', 'nds', 'nes', 'gba', 'ps1', 'ps2'}
        moved_any = False
        
        for item in os.listdir(dest_root):
            if item.startswith('.') or item in ('post_process.py', 'post_process.sh'):
                continue
            if item in platform_dirs:
                continue
            
            item_path = os.path.join(dest_root, item)
            platform = get_platform_for_item(item_path)
            if platform:
                target_dir = os.path.join(dest_root, platform)
                ensure_dir_and_chown(target_dir)
                
                dest_path = os.path.join(target_dir, item)
                print(f"Moving loose item '{item}' to '{dest_path}'...")
                move_and_chown(item_path, dest_path)
                moved_any = True
            else:
                print(f"Could not map loose item '{item}' to any platform.")
        
        if moved_any:
            trigger_retrom_update()

if __name__ == '__main__':
    main()
