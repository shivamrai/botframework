"""Environment setup helpers for the worker service."""
import os
import platform
import subprocess
import sys

def get_hardware_flags():
    """
    Detects hardware and returns the appropriate CMAKE_ARGS for llama-cpp-python.
    """
    system = platform.system()
    machine = platform.machine()

    flags = {}

    if system == "Darwin":
        if machine == "arm64":
            print("🍎 Detected Apple Silicon (Metal)")
            flags["CMAKE_ARGS"] = "-DLLAMA_METAL=on"
        else:
            print("💻 Detected Intel Mac (CPU Only)")
            flags["CMAKE_ARGS"] = "-DLLAMA_BLAS=off"  # Default to CPU

    elif system == "Linux":
        # Check for NVIDIA GPU
        try:
            subprocess.check_call(
                ["nvidia-smi"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
            print("🟢 Detected NVIDIA GPU (CUDA)")
            flags["CMAKE_ARGS"] = "-DLLAMA_CUBLAS=on"
        except (subprocess.CalledProcessError, FileNotFoundError):
            print("🐧 Detected Linux (CPU Only)")
            flags["CMAKE_ARGS"] = "-DLLAMA_BLAS=off"

    elif system == "Windows":
        # Basic check, can be improved
        print("🪟 Detected Windows")
        # Default to CPU for safety in this script.
        flags["CMAKE_ARGS"] = "-DLLAMA_BLAS=off"

    return flags

def setup_environment():
    """
    Sets up the virtual environment and installs dependencies.
    """
    base_dir = os.path.dirname(os.path.abspath(__file__))
    worker_dir = os.path.join(base_dir, "../worker")
    venv_dir = os.path.join(worker_dir, "venv")
    requirements_path = os.path.join(worker_dir, "requirements.txt")

    # 1. Create Virtual Environment
    if not os.path.exists(venv_dir):
        print(f"📦 Creating virtual environment at {venv_dir}...")
        subprocess.check_call([sys.executable, "-m", "venv", venv_dir])
    else:
        print(f"✅ Virtual environment exists at {venv_dir}")

    # 2. Determine Pip Path
    if platform.system() == "Windows":
        pip_exe = os.path.join(venv_dir, "Scripts", "pip.exe")
    else:
        pip_exe = os.path.join(venv_dir, "bin", "pip")

    # 3. Upgrade Pip
    print("⬆️  Upgrading pip...")
    subprocess.check_call([pip_exe, "install", "--upgrade", "pip"])

    # 4. Install Standard Dependencies (fastapi, uvicorn, etc.)
    print("📥 Installing standard dependencies...")
    subprocess.check_call([pip_exe, "install", "-r", requirements_path])

    # 5. Install llama-cpp-python with Hardware Acceleration
    print("🚀 Installing llama-cpp-python with hardware acceleration...")

    # Uninstall first to ensure clean rebuild if flags changed
    subprocess.call([pip_exe, "uninstall", "-y", "llama-cpp-python"])

    env = os.environ.copy()
    flags = get_hardware_flags()
    env.update(flags)

    # Force reinstall with --no-cache-dir to ensure compilation happens
    subprocess.check_call(
        [
            pip_exe,
            "install",
            "llama-cpp-python",
            "--force-reinstall",
            "--no-cache-dir",
        ],
        env=env,
    )

    print("\n✨ Setup Complete! ✨")
    print(
        "To run the worker manually:\n"
        f"source {venv_dir}/bin/activate && python {worker_dir}/main.py"
    )

if __name__ == "__main__":
    setup_environment()
