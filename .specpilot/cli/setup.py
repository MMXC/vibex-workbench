from setuptools import setup, find_packages
setup(
    name="specpilot",
    version="0.1.0",
    description="Agent-accessible prototype workspace manager",
    packages=find_packages(),
    install_requires=["pyyaml"],
    entry_points={
        "console_scripts": ["specpilot=specpilot.cli:main"],
    },
    python_requires=">=3.9",
)
