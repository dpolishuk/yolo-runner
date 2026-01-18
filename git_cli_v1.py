from __future__ import annotations


class GitCliAdapterV1:
    def __init__(self, command_runner, call_runner=None) -> None:
        self._command_runner = command_runner
        self._call_runner = call_runner or command_runner

    def add_all(self) -> None:
        self._call_runner(["git", "add", "."])

    def is_dirty(self) -> bool:
        output = self._command_runner(["git", "status", "--porcelain"], text=True)
        return bool(output.strip())

    def commit(self, message: str) -> None:
        self._call_runner(["git", "commit", "-m", message])

    def rev_parse_head(self) -> str:
        output = self._command_runner(["git", "rev-parse", "HEAD"], text=True)
        return output.strip()
