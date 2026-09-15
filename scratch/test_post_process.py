#!/usr/bin/env python3
import os
import sys
import shutil
import tempfile
import subprocess

def test_post_process():
    with tempfile.TemporaryDirectory() as test_dir:
        downloads_dir = os.path.join(test_dir, "downloads")
        roms_dir = os.path.join(test_dir, "roms")
        os.makedirs(downloads_dir, exist_ok=True)
        os.makedirs(roms_dir, exist_ok=True)

        print(f"Running tests with DOWNLOADS_DIR={downloads_dir} and ROMS_DIR={roms_dir}")
        
        os.environ["DOWNLOADS_DIR"] = downloads_dir
        os.environ["ROMS_DIR"] = roms_dir
        
        # 1. Create mock ROM files in the downloads directory
        rom_files = {
            "mario.sfc": "snes",
            "zelda.smc": "snes",
            "goldeneye.z64": "n64",
            "pokemon.nds": "nds",
            "metroid.nes": "nes",
            "mariokart.gba": "gba",
            "tekken.iso": "psx",
        }
        
        for filename in rom_files:
            file_path = os.path.join(downloads_dir, filename)
            with open(file_path, "wb") as f:
                f.write(b"mock rom content")
            print(f"Created mock file: {file_path}")

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

        # 3. Verify all files have been routed correctly into roms_dir/<platform>/
        for filename, expected_platform in rom_files.items():
            expected_path = os.path.join(roms_dir, expected_platform, filename)
            assert os.path.exists(expected_path), f"File {filename} was not moved to ROMs platform folder {expected_platform} (expected path: {expected_path})"
            print(f"Verified: {filename} correctly moved to {roms_dir}/{expected_platform}/")

        # 4. Verify no loose files are left in the downloads directory
        loose_files = [f for f in os.listdir(downloads_dir) if os.path.isfile(os.path.join(downloads_dir, f))]
        assert len(loose_files) == 0, f"Loose files remaining in downloads directory: {loose_files}"
        print("Verified: No loose files left in downloads directory.")

    print("\nAll tests PASSED successfully!")

if __name__ == "__main__":
    test_post_process()
