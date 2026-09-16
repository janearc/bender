#!/usr/bin/env python3
# Entry point so `python3 -m exposure` prints the standards. Without it runpy
# imports the package and then re-executes the module, which works but warns.

import sys

from .standards import main

if __name__ == "__main__":
    sys.exit(main())
