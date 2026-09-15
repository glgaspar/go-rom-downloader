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
        
        # 1. Create mock files matching user's exact folder structure
        # Loose files in downloads
        decap_path = os.path.join(downloads_dir, "Decap Attack.bin")
        with open(decap_path, "wb") as f:
            f.write(b"mock megadrive bin content")

        kid_path = os.path.join(downloads_dir, "Kid Chameleon.bin")
        with open(kid_path, "wb") as f:
            f.write(b"mock megadrive bin content")

        readme_path = os.path.join(downloads_dir, "readme.html")
        with open(readme_path, "wb") as f:
            f.write(b"<html>readme</html>")

        # Nested platform folders inside downloads
        dl_n64_dir = os.path.join(downloads_dir, "n64")
        os.makedirs(dl_n64_dir, exist_ok=True)
        diddy_path = os.path.join(dl_n64_dir, "Diddy Kong Racing (USA) (En,Fr).n64")
        with open(diddy_path, "wb") as f:
            f.write(b"mock n64 content")

        dl_ps2_dir = os.path.join(downloads_dir, "ps2")
        os.makedirs(dl_ps2_dir, exist_ok=True)
        naruto_path = os.path.join(dl_ps2_dir, "Naruto - Uzumaki Chronicles 2.7z")
        with open(naruto_path, "wb") as f:
            f.write(b"mock ps2 7z content")

        # Pre-existing organized roms in ROMs library
        os.makedirs(os.path.join(roms_dir, "gba"), exist_ok=True)
        with open(os.path.join(roms_dir, "gba", "Pokémon Emerald Version (U)(TrashMan).gba"), "wb") as f:
            f.write(b"existing gba rom")

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
        assert os.path.exists(os.path.join(roms_dir, "megadrive", "Decap Attack.bin")), "Decap Attack.bin was not moved to megadrive/"
        assert os.path.exists(os.path.join(roms_dir, "megadrive", "Kid Chameleon.bin")), "Kid Chameleon.bin was not moved to megadrive/"
        assert os.path.exists(os.path.join(roms_dir, "n64", "Diddy Kong Racing (USA) (En,Fr).n64")), "Diddy Kong Racing was not moved to n64/"
        assert os.path.exists(os.path.join(roms_dir, "ps2", "Naruto - Uzumaki Chronicles 2.7z")), "Naruto 7z was not moved to ps2/"
        
        # Verify readme.html remains in downloads
        assert os.path.exists(readme_path), "readme.html should remain in downloads"
        
        # Verify empty subdirectories in downloads were cleaned up
        assert not os.path.exists(dl_n64_dir), "empty downloads/n64 dir should be cleaned up"
        assert not os.path.exists(dl_ps2_dir), "empty downloads/ps2 dir should be cleaned up"

    print("\nAll tests PASSED successfully!")

if __name__ == "__main__":
    test_post_process()
