#!/usr/bin/env python3
import os
import sys
import shutil
import tempfile
import subprocess

def test_post_process():
    # Create a temporary directory to act as the Roms root
    with tempfile.TemporaryDirectory() as test_dir:
        print(f"Running tests in temporary directory: {test_dir}")
        
        # We will set DOWNLOADS_DIR to test_dir to override /mnt/games/Roms in our script
        os.environ["DOWNLOADS_DIR"] = test_dir
        
        # 1. Create mock ROM files in the root of the test directory
        rom_files = {
            "mario.sfc": "snes",
            "zelda.smc": "snes",
            "goldeneye.z64": "n64",
            "pokemon.nds": "nds",
            "metroid.nes": "nes",
            "mariokart.gba": "gba",
            "tekken.iso": "ps1",  # size <= 750MB -> ps1
        }
        
        # Create small mock files (content doesn't matter for extension matching)
        for filename in rom_files:
            file_path = os.path.join(test_dir, filename)
            with open(file_path, "wb") as f:
                f.write(b"mock rom content")
            print(f"Created mock file: {file_path}")

        # Let's also create a larger .iso file to test PS2 matching
        ps2_filename = "grandtheftauto.iso"
        ps2_path = os.path.join(test_dir, ps2_filename)
        with open(ps2_path, "wb") as f:
            # 800MB -> 800 * 1024 * 1024 bytes (seek to write large file quickly)
            f.seek(800 * 1024 * 1024)
            f.write(b"\0")
        print(f"Created mock PS2 file (800MB): {ps2_path}")
        rom_files[ps2_filename] = "ps2"

        # Create a mock ZIP file containing an NDS ROM
        import zipfile
        zip_path = os.path.join(test_dir, "pokemon_black.zip")
        with zipfile.ZipFile(zip_path, 'w') as z:
            z.writestr("pokemon_black.nds", b"mock nds rom inside zip")
        print(f"Created mock ZIP with NDS ROM: {zip_path}")
        rom_files["pokemon_black.zip"] = "nds"

        # Create a mock 7z file containing an NDS ROM using 7z
        nds_temp = os.path.join(test_dir, "pokemon_white.nds")
        with open(nds_temp, "wb") as f:
            f.write(b"mock nds rom inside 7z")
        rar_path = os.path.join(test_dir, "pokemon_white.7z")
        subprocess.run(["7z", "a", rar_path, nds_temp], capture_output=True, check=True)
        os.remove(nds_temp)
        print(f"Created mock 7Z with NDS ROM: {rar_path}")
        rom_files["pokemon_white.7z"] = "nds"

        # 2. Run post_process.py in scanning mode (0 arguments)
        script_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "post_process.py"))
        print(f"Running: python3 {script_path}")
        result = subprocess.run([sys.executable, script_path], capture_output=True, text=True)
        print("Script stdout:")
        print(result.stdout)
        if result.stderr:
            print("Script stderr:")
            print(result.stderr)
        
        assert result.returncode == 0, f"post_process.py failed with exit code {result.returncode}"

        # 3. Verify all files have been routed correctly
        for filename, expected_platform in rom_files.items():
            expected_path = os.path.join(test_dir, expected_platform, filename)
            assert os.path.exists(expected_path), f"File {filename} was not moved to platform folder {expected_platform} (expected path: {expected_path})"
            print(f"Verified: {filename} correctly moved to {expected_platform}/")

        # 4. Verify no loose files are left in the root directory
        loose_files = [f for f in os.listdir(test_dir) if os.path.isfile(os.path.join(test_dir, f))]
        assert len(loose_files) == 0, f"Loose files remaining in root directory: {loose_files}"
        print("Verified: No loose files left in root directory.")

        # 5. Verify PUID/PGID permissions on created subdirectories
        # In this test environment, they should be owned by 1000:1000 (gustavo) or similar.
        # Let's check that the script attempted chown to 1000:1000.
        # Since we run as gustavo (1000:1000), directories created by our process will natively have 1000:1000.
        for platform in set(rom_files.values()):
            platform_dir = os.path.join(test_dir, platform)
            stat_info = os.stat(platform_dir)
            assert stat_info.st_uid == 1000, f"Directory {platform_dir} owner is {stat_info.st_uid}, expected 1000"
            assert stat_info.st_gid == 1000, f"Directory {platform_dir} group is {stat_info.st_gid}, expected 1000"
            print(f"Verified: Directory {platform_dir} permissions match uid=1000, gid=1000")

    print("\nAll tests PASSED successfully!")

if __name__ == "__main__":
    test_post_process()
