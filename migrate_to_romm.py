#!/usr/bin/env python3
import os
import sys
import shutil

def merge_folders(src, dst):
    """Safely merges files from src to dst and removes src once empty."""
    if not os.path.exists(dst):
        os.makedirs(dst, exist_ok=True)
        try:
            # Maintain standard PUID/PGID permissions (1000:1000)
            os.chown(dst, 1000, 1000)
        except Exception:
            pass

    for item in os.listdir(src):
        s = os.path.join(src, item)
        d = os.path.join(dst, item)
        if os.path.isdir(s):
            merge_folders(s, d)
        else:
            if not os.path.exists(d):
                print(f"  Moving file: {os.path.basename(s)} -> {os.path.dirname(d)}")
                shutil.move(s, d)
                try:
                    os.chown(d, 1000, 1000)
                except Exception:
                    pass
            else:
                print(f"  Skipping duplicate file: {d}")
    
    try:
        os.rmdir(src)
    except OSError:
        pass

def main():
    dest_root = os.environ.get("DOWNLOADS_DIR")
    if not dest_root:
        dest_root = "/mnt/games/Roms"
        if not os.path.exists(dest_root):
            dest_root = "./downloads"
    
    dest_root = os.path.abspath(dest_root)
    if not os.path.exists(dest_root):
        print(f"Error: Target directory '{dest_root}' does not exist.")
        sys.exit(1)

    print(f"Starting migration to RomM structure in: {dest_root}")

    # 1. Rename ps1 to psx if it exists in the root
    ps1_dir = os.path.join(dest_root, "ps1")
    psx_dir = os.path.join(dest_root, "psx")
    if os.path.exists(ps1_dir):
        print(f"Found legacy PlayStation 1 directory: {ps1_dir}")
        merge_folders(ps1_dir, psx_dir)
        print(f"Merged PS1 files into {psx_dir}")

    # 2. Create the recommended 'roms' subfolder under the target root
    roms_root = os.path.join(dest_root, "roms")
    os.makedirs(roms_root, exist_ok=True)
    try:
        os.chown(roms_root, 1000, 1000)
    except Exception:
        pass

    # 3. Move all platform directories into the 'roms' folder
    platform_dirs = {'snes', 'n64', 'nds', 'nes', 'gba', 'psx', 'ps2'}
    
    moved_count = 0
    for platform in platform_dirs:
        platform_src = os.path.join(dest_root, platform)
        platform_dst = os.path.join(roms_root, platform)
        
        # If there's an old 'ps1' directory that didn't get merged or is inside,
        # let's check and merge it directly into the roms/psx destination as well.
        if platform == 'psx' and os.path.exists(os.path.join(dest_root, "ps1")):
            platform_src = os.path.join(dest_root, "ps1")

        if os.path.exists(platform_src):
            print(f"Migrating platform folder '{os.path.basename(platform_src)}' to '{platform_dst}'...")
            merge_folders(platform_src, platform_dst)
            moved_count += 1

    print("\nMigration completed successfully!")
    print(f"Migrated {moved_count} platform folders under the new 'roms/' structure.")
    print(f"Your ROMs library root is located at: {dest_root}")
    print("\nNext Steps:")
    print("1. Set ROMM_API_ADDR and ROMM_API_KEY environment variables in docker-compose.yml.")
    print("2. Re-mount your host volume in RomM docker configuration to point to this directory.")
    print("3. Future downloads organized by the postprocessor will automatically go into the 'roms/' folder.")

if __name__ == '__main__':
    main()
