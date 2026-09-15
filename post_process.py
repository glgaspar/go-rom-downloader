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
        '.v64': 'n64',
        '.nds': 'nds',
        '.nes': 'nes',
        '.gba': 'gba',
        '.gbc': 'gbc',
        '.gb': 'gb',
        '.cso': 'psp',
        '.pbp': 'psp',
        '.3ds': '3ds',
        '.cia': '3ds',
        '.md': 'megadrive',
        '.gen': 'megadrive',
        '.smd': 'megadrive',
        '.iso': 'disc_based',
        '.chd': 'disc_based',
        '.rvz': 'disc_based',
        '.gcm': 'gamecube',
        '.wbfs': 'wii',
    }

    platform = EXTENSION_MAP.get(ext)
    c_name = (console_name or "").lower()
    f_name = filename.lower()

    if not platform:
        if 'snes' in c_name or 'super nintendo' in c_name:
            platform = 'snes'
        elif 'nintendo 64' in c_name or 'n64' in c_name:
            platform = 'n64'
        elif 'game boy advance' in c_name or 'gba' in c_name:
            platform = 'gba'
        elif 'game boy color' in c_name or 'gbc' in c_name:
            platform = 'gbc'
        elif 'game boy' in c_name:
            platform = 'gb'
        elif 'nintendo ds' in c_name or 'nds' in c_name:
            platform = 'nds'
        elif 'nintendo 3ds' in c_name or '3ds' in c_name:
            platform = '3ds'
        elif 'nes' in c_name or 'nintendo entertainment system' in c_name:
            platform = 'nes'
        elif 'genesis' in c_name or 'mega drive' in c_name or 'megadrive' in c_name:
            platform = 'megadrive'
        else:
            return None

    if platform == 'disc_based':
        if 'playstation 2' in c_name or 'ps2' in c_name or 'playstation 2' in f_name or 'ps2' in f_name:
            return 'ps2'
        elif 'psp' in c_name or 'playstation portable' in c_name or 'psp' in f_name:
            return 'psp'
        elif 'gamecube' in c_name or 'ngc' in c_name or 'gamecube' in f_name:
            return 'gamecube'
        elif 'wii' in c_name or 'wii' in f_name:
            return 'wii'
        elif 'playstation' in c_name or 'psx' in c_name or 'ps1' in c_name or 'playstation' in f_name or 'psx' in f_name or 'ps1' in f_name:
            return 'psx'
        else:
            # Fallback based on size: PS1 <= 750MB, PS2 > 750MB
            if size > 786432000:
                return 'ps2'
            else:
                return 'psx'

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

def trigger_romm_update():
    api_addr = os.environ.get("ROMM_API_ADDR") or os.environ.get("ROMM_URL")
    api_key = os.environ.get("ROMM_API_KEY")

    if not api_addr or not api_key:
        print("ROMM_API_ADDR/ROMM_URL or ROMM_API_KEY environment variables are not set. Skipping library update.")
        return

    api_addr = api_addr.rstrip('/')
    url = f"{api_addr}/api/tasks/run/scan_library"

    print(f"Triggering RomM library update at {url}...")
    
    import urllib.request
    import urllib.error

    req = urllib.request.Request(url, method='POST')
    req.add_header('Authorization', f'Bearer {api_key}')

    try:
        with urllib.request.urlopen(req, data=b'') as response:
            status = response.status
            body = response.read().decode('utf-8')
            print(f"RomM library update triggered successfully. Status: {status}")
            if body:
                print(f"Response: {body}")
    except urllib.error.HTTPError as e:
        print(f"Failed to trigger RomM library update (HTTP {e.code}): {e.read().decode('utf-8', errors='ignore')}")
    except Exception as e:
        print(f"Error triggering RomM library update: {e}")
def main():
    roms_dir = os.environ.get("ROMS_DIR")
    downloads_dir = os.environ.get("DOWNLOADS_DIR")

    if roms_dir:
        dest_root = roms_dir
    elif downloads_dir and os.path.isdir(os.path.join(downloads_dir, "roms")):
        dest_root = os.path.join(downloads_dir, "roms")
    elif downloads_dir:
        dest_root = downloads_dir
    elif os.path.exists("/mnt/games/Roms"):
        dest_root = "/mnt/games/Roms"
    else:
        dest_root = "./roms"
    
    dest_root = os.path.abspath(dest_root)
    ensure_dir_and_chown(dest_root)
    print(f"Target ROMs library directory: {dest_root}")

    platform_dirs = {'snes', 'n64', 'nds', 'nes', 'gba', 'gbc', 'gb', 'psx', 'ps2', 'psp', '3ds', 'megadrive', 'gamecube', 'wii'}

    # Case 1: Specific path passed as argument
    if len(sys.argv) > 1:
        item_path = sys.argv[1]
        console_name = sys.argv[2] if len(sys.argv) > 2 else None
        
        if not os.path.exists(item_path):
            search_dirs = [d for d in [downloads_dir, dest_root] if d and os.path.exists(d)]
            found = False
            for d in search_dirs:
                alternative_path = os.path.join(d, os.path.basename(item_path))
                if os.path.exists(alternative_path):
                    item_path = alternative_path
                    found = True
                    break
            if not found:
                print(f"Error: Path '{item_path}' does not exist.")
                sys.exit(1)

        item_path = os.path.abspath(item_path)
        
        # Don't process if it's already inside a platform subfolder
        parent_dir = os.path.basename(os.path.dirname(item_path))
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
            trigger_romm_update()
        else:
            print(f"No platform matched for item: '{item_path}'")
    
    # Case 2: Scan directories for loose files
    else:
        scan_dirs = []
        if downloads_dir and os.path.isdir(downloads_dir):
            scan_dirs.append(os.path.abspath(downloads_dir))
        if dest_root not in scan_dirs and os.path.isdir(dest_root):
            scan_dirs.append(dest_root)

        moved_any = False
        for s_dir in scan_dirs:
            print(f"Scanning '{s_dir}' for loose ROMs/games...")
            for item in os.listdir(s_dir):
                if item.startswith('.') or item in ('post_process.py', 'post_process.sh'):
                    continue
                if item in platform_dirs or item == 'roms':
                    continue
                
                item_path = os.path.join(s_dir, item)
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
            trigger_romm_update()

if __name__ == '__main__':
    main()
