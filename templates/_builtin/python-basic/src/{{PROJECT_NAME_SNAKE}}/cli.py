"""Command line interface of {{PROJECT_NAME}}."""

import argparse
from typing import Optional, Sequence

from . import __version__


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="{{PROJECT_NAME_KEBAB}}", description="{{PROJECT_NAME}}")
    parser.add_argument("--name", default="world", help="who to greet")
    parser.add_argument("--version", action="version", version=f"%(prog)s {__version__}")
    return parser


def main(argv: Optional[Sequence[str]] = None) -> int:
    args = build_parser().parse_args(argv)
    print(f"Hello, {args.name}! This is {{PROJECT_NAME}}.")
    return 0
