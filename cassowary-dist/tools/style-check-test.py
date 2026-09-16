#!/usr/bin/env python3
# style-check-test -- the hook against a crafted corpus.
#
# Runs style-check.py as a subprocess, one hook payload per case, and checks
# that it blocks what it should and passes what it should. the negatives
# matter as much as the positives: bounded fault claims, estate-register
# prose, capitals inside string literals and code fences, and short notes
# must all pass, because a hook that fires on healthy writing gets turned
# off, and then it catches nothing.
#
#   style-check-test.py
#
# Prints one row per case and a final count. exits with the number of cases
# that misbehaved, so zero means the hook does what this file says it does.

import json
import os
import subprocess
import sys

HOOK = os.path.join(os.path.dirname(os.path.abspath(__file__)), "style-check.py")

# sixty words of plain register, reused as filler so density cases rest on a
# realistic denominator rather than a handful of words
FILLER = (
    "the audit walks the cluster and checks each workload in turn. "
    "it carries no inventory, so a change in the cluster does not invalidate "
    "the tool, it changes the report. the table names each workload once and "
    "the number at the end is the count that remains. the freeze ends when "
    "that number reaches zero and not before. ")


# one write payload, the common case
def write(path, content):
    return {"tool_name": "Write",
            "tool_input": {"file_path": path, "content": content}}


# one edit payload
def edit(path, new_string):
    return {"tool_name": "Edit",
            "tool_input": {"file_path": path, "new_string": new_string}}


# one bash payload
def bash(command):
    return {"tool_name": "Bash", "tool_input": {"command": command}}


# name, payload, expected outcome. 'allow' cases are the known negatives that
# must not fire; 'deny' cases are the condition the hook exists to refuse
CASES = [
    ("bounded fault in markdown", write("/tmp/notes.md",
        FILLER + "i got the tag parsing wrong in `image_parts`, fixed in "
        "commit abc1234, and the audit now agrees with the cluster."),
     "allow"),

    ("estate register with rfc 2119 words", write("/tmp/doc.md",
        "the service MUST publish health and SHOULD report its own sha. " +
        FILLER + "a **single** emphasis is ordinary writing, not a signal."),
     "allow"),

    ("capitals in string literals, comments clean", write("/tmp/app.py",
        '# format one row for the operator\n'
        'BANNER = "CRITICAL WARNING FAILURE URGENT BROKEN"\n'
        'LEVELS = ["SEVERE", "ALARMING", "DISASTROUS"]\n'
        '# the operator reads the banner, the hook reads this line\n'
        'def render(row):\n'
        '    return BANNER + row\n'),
     "allow"),

    ("short edit below the word floor", edit("/tmp/notes.md",
        "NOTE the WATCHDOG restarts the poller"),
     "allow"),

    ("bash that is not a commit", bash(
        'echo "i was wrong about everything and i am sorry"'),
     "allow"),

    ("clean commit message", bash(
        'git commit -m "freeze-audit: report the count, not the judgement" '
        '-m "Co-Authored-By: Claude <claude-fable-5> <noreply@anthropic.com>"'),
     "allow"),

    ("capitals fenced off in markdown", write("/tmp/doc.md",
        FILLER + "\n\n```\nEXPORT PATH MODE FLAGS RETRY BANNER LEVELS\n"
        "SEVERE ALARMING DISASTROUS URGENT BROKEN\n```\n"),
     "allow"),

    ("fault words with no first person", write("/tmp/doc.md",
        FILLER + "the test failed on line 40 and the fix is in commit abc1234."),
     "allow"),

    ("json passes untouched", write("/tmp/data.json",
        '{"level": "CRITICAL", "note": "i am sorry"}'),
     "allow"),

    ("unbounded fault in markdown", write("/tmp/handoff.md",
        FILLER + "i keep getting this wrong. i am sorry. i should have "
        "known better and i did not."),
     "deny"),

    ("capitals at peer-message density", write("/tmp/msg.md",
        "IMPORTANT the poller is BROKEN and the queue is DRAINING. " + FILLER +
        "NEVER restart the worker BEFORE the drain completes, ALWAYS wait. "
        "this is URGENT and the WATCHDOG will not save you."),
     "deny"),

    ("bold past the limit", write("/tmp/doc.md",
        "**read this first** and then **do not skip** the drain, because "
        "**the queue** will not survive a **cold restart** either. " + FILLER),
     "deny"),

    ("session trailer in a commit", bash(
        'git commit -m "fix the poller" '
        '-m "Claude-Session: https://claude.ai/code/session_abc123"'),
     "deny"),

    ("session trailer in a heredoc commit", bash(
        'git commit -m "$(cat <<\'EOF\'\nfix the poller\n\n'
        'Claude-Session: https://claude.ai/code/session_abc123\nEOF\n)"'),
     "deny"),

    ("unbounded fault in a source comment", write("/tmp/app.py",
        '# i broke this again and i am sorry\n'
        'def render(row):\n'
        '    return row\n'),
     "deny"),

    ("unbounded fault in a commit message", bash(
        'git commit -m "i overstated what was checked and i am sorry '
        'about the noise"'),
     "deny"),
]


# run one payload through the hook and report which way it went
def outcome(payload):
    r = subprocess.run([sys.executable, HOOK], input=json.dumps(payload),
                       capture_output=True, text=True, timeout=30)
    return "deny" if r.returncode == 2 else "allow", r.stderr.strip()


# every case, one row each, then the count that misbehaved
def main():
    bad = 0
    print(f"{'case':<44}{'expected':<10}{'got':<8}{'verdict':<8}")
    print("-" * 68)
    for name, payload, expected in CASES:
        got, stderr = outcome(payload)
        ok = got == expected
        bad += 0 if ok else 1
        print(f"{name:<44}{expected:<10}{got:<8}{'ok' if ok else 'bad':<8}")
        if not ok and stderr:
            print(f"    {stderr}")
    print()
    print(f"{len(CASES) - bad} of {len(CASES)} cases behave as specified")
    return bad


if __name__ == "__main__":
    sys.exit(main())
