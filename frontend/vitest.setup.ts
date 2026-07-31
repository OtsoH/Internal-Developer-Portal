import "@testing-library/jest-dom/vitest";

import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

// Testing Library only auto-cleans when Vitest runs with `globals: true`, which
// this config does not. Without it, a second `render` in the same file leaves
// the first one mounted and every query finds two matches.
afterEach(cleanup);
