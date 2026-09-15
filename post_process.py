#!/usr/bin/env python3
import os
import sys
import shutil
import zipfile
import subprocess

def log(msg):
    print(msg, flush=True)

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
        log(f"  [ZIP-ERROR] Error reading zip {zip_path}: {e}")
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

    # Extension to Platform Mapping
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
        '.md': 'megadrive',
        '.gen': 'megadrive',
        '.smd': 'megadrive',
        '.sms': 'mastersystem',
        '.gg': 'gamegear',
        '.cso': 'psp',
        '.pbp': 'psp',
        '.3ds': '3ds',
        '.cia': '3ds',
        '.iso': 'disc_based',
        '.chd': 'disc_based',
        '.rvz': 'disc_based',
        '.gcm': 'gamecube',
        '.wbfs': 'wii',
        '.bin': 'ambiguous_bin',
    }

    platform = EXTENSION_MAP.get(ext)
    c_name = (console_name or "").lower()
    f_name = filename.lower()

    if platform == 'ambiguous_bin':
        if 'genesis' in c_name or 'mega drive' in c_name or 'megadrive' in c_name or 'sega' in c_name:
            return 'megadrive'
        elif 'psx' in c_name or 'ps1' in c_name or 'playstation' in c_name or 'psx' in f_name or 'ps1' in f_name:
            return 'psx'
        elif size > 33554432: # > 32MB usually PS1
            return 'psx'
        else:
            return 'megadrive'

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
        elif 'ps2' in c_name or 'playstation 2' in c_name:
            platform = 'ps2'
        elif 'psx' in c_name or 'ps1' in c_name or 'playstation' in c_name:
            platform = 'psx'
        elif 'psp' in c_name or 'playstation portable' in c_name:
            platform = 'psp'
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
            log(f"  [WARN] chown for {path} failed: {e}")

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
        log(f"  [WARN] chown for {dest} failed: {e}")

def remove_empty_dirs(root_dir):
    for root, dirs, files in os.walk(root_dir, topdown=False):
        for d in dirs:
            dir_path = os.path.join(root, d)
            try:
                if not os.listdir(dir_path):
                    os.rmdir(dir_path)
                    log(f"  [CLEANUP] Removed empty directory: {dir_path}")
            except Exception:
                pass

def trigger_romm_update():
    api_addr = os.environ.get("ROMM_API_ADDR") or os.environ.get("ROMM_URL")
    api_key = os.environ.get("ROMM_API_KEY")

    if not api_addr or not api_key:
        log("[ROMM UPDATE] ROMM_API_ADDR or ROMM_API_KEY env vars not set. Skipping library scan trigger.")
        return

    api_addr = api_addr.rstrip('/')
    url = f"{api_addr}/api/tasks/run/scan_library"

    log(f"[ROMM UPDATE] Triggering RomM library update at {url}...")
    import urllib.request
    import urllib.error

    req = urllib.request.Request(url, method='POST')
    req.add_header('Authorization', f'Bearer {api_key}')

    try:
        with urllib.request.urlopen(req, data=b'') as response:
            status = response.status
            body = response.read().decode('utf-8')
            log(f"[ROMM UPDATE] Triggered successfully. Status: {status}")
            if body:
                log(f"[ROMM UPDATE] Response: {body}")
    except urllib.error.HTTPError as e:
        log(f"[ROMM UPDATE] Failed to trigger (HTTP {e.code}): {e.read().decode('utf-8', errors='ignore')}")
    except Exception as e:
        log(f"[ROMM UPDATE] Error: {e}")

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

    log("==================================================")
    log("[ROM ORGANIZER] Starting post-processing organization...")
    log(f"[CONFIG] Downloads Staging: {downloads_dir}")
    log(f"[CONFIG] Target ROMs Library: {dest_root}")
    log("==================================================")

    platform_dirs = {'snes', 'n64', 'nds', 'nes', 'gba', 'gbc', 'gb', 'psx', 'ps2', 'psp', '3ds', 'megadrive', 'mastersystem', 'gamegear', 'gamecube', 'wii'}

    # Case 1: Specific path passed as argument
    if len(sys.argv) > 1:
        item_path = sys.argv[1]
        console_name = sys.argv[2] if len(sys.argv) > 2 else None
        log(f"[MODE] Processing specific item: '{item_path}' (Console Hint: '{console_name}')")

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
                log(f"[ERROR] Path '{item_path}' does not exist.")
                sys.exit(1)

        item_path = os.path.abspath(item_path)
        parent_dir = os.path.basename(os.path.dirname(item_path))

        if parent_dir in platform_dirs and os.path.dirname(item_path) != downloads_dir:
            log(f"[SKIP] Item '{item_path}' is already inside platform folder '{parent_dir}'.")
            sys.exit(0)

        platform = get_platform_for_item(item_path, console_name or parent_dir)
        if platform:
            target_dir = os.path.join(dest_root, platform)
            ensure_dir_and_chown(target_dir)
            dest_path = os.path.join(target_dir, os.path.basename(item_path))
            log(f"[MOVE] Moving '{item_path}' -> '{dest_path}'")
            move_and_chown(item_path, dest_path)
            trigger_romm_update()
        else:
            log(f"[WARN] No platform matched for item: '{item_path}'")

    # Case 2: Full scan of downloads staging & loose files
    else:
        log("[MODE] Running full directory scan...")
        moved_count = 0

        # Sub-case 2a: Scan downloads_dir recursively
        if downloads_dir and os.path.isdir(downloads_dir):
            abs_downloads = os.path.abspath(downloads_dir)
            log(f"[SCAN] Walking downloads directory: '{abs_downloads}'")

            for root, dirs, files in os.walk(abs_downloads):
                dirs[:] = [d for d in dirs if not d.startswith('.')]
                
                for file_name in files:
                    if file_name.startswith('.') or file_name in ('post_process.py', 'post_process.sh', 'readme.html', 'readme.txt', 'readme.md'):
                        continue
                    
                    file_ext = os.path.splitext(file_name)[1].lower()
                    if file_ext in ('.html', '.txt', '.nfo', '.jpg', '.png', '.jpeg', '.svg', '.db'):
                        continue

                    full_path = os.path.join(root, file_name)
                    
                    rel_dir = os.path.relpath(root, abs_downloads)
                    parent_hint = None
                    if rel_dir != '.':
                        parent_hint = rel_dir.split(os.sep)[0]

                    platform = get_platform_for_item(full_path, console_name=parent_hint)
                    if platform:
                        target_dir = os.path.join(dest_root, platform)
                        ensure_dir_and_chown(target_dir)
                        dest_path = os.path.join(target_dir, file_name)
                        log(f"[MOVE] '{full_path}' -> '{dest_path}' (Platform: {platform})")
                        move_and_chown(full_path, dest_path)
                        moved_count += 1
                    else:
                        log(f"[SKIP] Could not map loose file '{full_path}' to any platform.")

            remove_empty_dirs(abs_downloads)

        # Sub-case 2b: Scan root of dest_root for loose files
        if os.path.isdir(dest_root):
            log(f"[SCAN] Checking root of ROMs directory for unorganized files: '{dest_root}'")
            for item in os.listdir(dest_root):
                if item.startswith('.') or item in ('post_process.py', 'post_process.sh', 'roms'):
                    continue
                if item in platform_dirs:
                    continue

                item_path = os.path.join(dest_root, item)
                platform = get_platform_for_item(item_path)
                if platform:
                    target_dir = os.path.join(dest_root, platform)
                    ensure_dir_and_chown(target_dir)
                    dest_path = os.path.join(target_dir, item)
                    log(f"[MOVE] Loose root item '{item_path}' -> '{dest_path}' (Platform: {platform})")
                    move_and_chown(item_path, dest_path)
                    moved_count += 1

        log(f"[SUMMARY] Total files moved and organized: {moved_count}")
        log("==================================================")
        if moved_count > 0:
            trigger_romm_update()

if __name__ == '__main__':
    main()
