import unittest

from git_cli_v1 import GitCliAdapterV1


class GitCliAdapterV1Tests(unittest.TestCase):
    def test_is_dirty_false_when_porcelain_empty(self):
        adapter = GitCliAdapterV1(command_runner=lambda args, text=True: "")

        self.assertFalse(adapter.is_dirty())

    def test_is_dirty_true_when_porcelain_has_output(self):
        adapter = GitCliAdapterV1(command_runner=lambda args, text=True: " M file.txt\n")

        self.assertTrue(adapter.is_dirty())

    def test_is_dirty_propagates_errors(self):
        def failing_runner(args, text=True):
            raise RuntimeError("git failed")

        adapter = GitCliAdapterV1(command_runner=failing_runner)

        with self.assertRaises(RuntimeError):
            adapter.is_dirty()

    def test_commit_runs_git_commit(self):
        calls = []

        def call_runner(args):
            calls.append(args)

        adapter = GitCliAdapterV1(command_runner=lambda args, text=True: "", call_runner=call_runner)

        adapter.commit("hello")

        self.assertEqual(calls, [["git", "commit", "-m", "hello"]])

    def test_rev_parse_head_runs_git_rev_parse(self):
        seen_args = []

        def command_runner(args, text=True):
            seen_args.append(args)
            return "abc123\n"

        adapter = GitCliAdapterV1(command_runner=command_runner)

        self.assertEqual(adapter.rev_parse_head(), "abc123")
        self.assertEqual(seen_args, [["git", "rev-parse", "HEAD"]])

    def test_add_all_runs_git_add(self):
        calls = []

        def call_runner(args):
            calls.append(args)

        adapter = GitCliAdapterV1(command_runner=lambda args, text=True: "", call_runner=call_runner)

        adapter.add_all()

        self.assertEqual(calls, [["git", "add", "."]])


if __name__ == "__main__":
    unittest.main()
