import json
import unittest

from beads_yolo_runner import select_first_open_leaf_task_id


def make_command_runner(children_map: dict[str, list[dict]]):
    def runner(args, text=False):
        if args[:3] != ["bd", "ready", "--parent"] or len(args) < 5:
            raise AssertionError(f"Unexpected command: {args}")
        parent_id = args[3]
        return json.dumps(children_map.get(parent_id, []))

    return runner


class SelectFirstOpenLeafTaskIDTests(unittest.TestCase):
    def test_returns_direct_task_leaf(self):
        command_runner = make_command_runner(
            {
                "root": [
                    {
                        "id": "task-1",
                        "issue_type": "task",
                        "status": "open",
                        "priority": 1,
                    }
                ]
            }
        )

        self.assertEqual(select_first_open_leaf_task_id("root", command_runner), "task-1")

    def test_recurses_into_epic_for_first_open_leaf(self):
        command_runner = make_command_runner(
            {
                "root": [
                    {
                        "id": "epic-1",
                        "issue_type": "epic",
                        "status": "open",
                        "priority": 1,
                    },
                    {
                        "id": "task-2",
                        "issue_type": "task",
                        "status": "open",
                        "priority": 2,
                    },
                ],
                "epic-1": [
                    {
                        "id": "task-1",
                        "issue_type": "task",
                        "status": "open",
                        "priority": 1,
                    }
                ],
            }
        )

        self.assertEqual(select_first_open_leaf_task_id("root", command_runner), "task-1")

    def test_skips_non_open_children(self):
        command_runner = make_command_runner(
            {
                "root": [
                    {
                        "id": "task-1",
                        "issue_type": "task",
                        "status": "closed",
                        "priority": 1,
                    },
                    {
                        "id": "task-2",
                        "issue_type": "task",
                        "status": "open",
                        "priority": 2,
                    },
                ]
            }
        )

        self.assertEqual(select_first_open_leaf_task_id("root", command_runner), "task-2")

    def test_missing_priority_sorts_last(self):
        command_runner = make_command_runner(
            {
                "root": [
                    {
                        "id": "task-1",
                        "issue_type": "task",
                        "status": "open",
                        "priority": 1000,
                    },
                    {
                        "id": "task-2",
                        "issue_type": "task",
                        "status": "open",
                    },
                ]
            }
        )

        self.assertEqual(select_first_open_leaf_task_id("root", command_runner), "task-1")

    def test_empty_children_returns_none(self):
        command_runner = make_command_runner(
            {
                "root": [
                    {
                        "id": "epic-1",
                        "issue_type": "epic",
                        "status": "open",
                        "priority": 1,
                    }
                ],
                "epic-1": [],
            }
        )

        self.assertIsNone(select_first_open_leaf_task_id("root", command_runner))


if __name__ == "__main__":
    unittest.main()
