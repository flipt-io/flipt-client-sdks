import pathlib
import tomllib


def test_package_requires_python_3_11_or_newer():
    pyproject_path = pathlib.Path(__file__).parents[1] / "pyproject.toml"
    pyproject = tomllib.loads(pyproject_path.read_text())

    assert pyproject["tool"]["poetry"]["dependencies"]["python"] == ">=3.11,<4.0"
