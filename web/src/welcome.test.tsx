import { afterEach, expect, test } from "bun:test";
import { cleanup, render, screen } from "@testing-library/react";
import { Welcome } from "./Welcome";

afterEach(cleanup);

test("protected screen shows the welcome sentence", () => {
  render(<Welcome email="ada@example.com" />);
  expect(
    screen.getByText("Hello ada@example.com, welcome back").textContent,
  ).toBe("Hello ada@example.com, welcome back");
});
