import { describe, it, expect } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useActionState } from "./useActionState";

describe("useActionState", () => {
  it("starts idle", () => {
    const { result } = renderHook(() => useActionState());
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("sets loading during execution", async () => {
    let resolve!: () => void;
    const promise = new Promise<void>((r) => {
      resolve = r;
    });

    const { result } = renderHook(() => useActionState());

    act(() => {
      result.current.execute(() => promise);
    });

    expect(result.current.loading).toBe(true);
    expect(result.current.error).toBeNull();

    await act(async () => {
      resolve();
    });

    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("captures error on failure", async () => {
    const { result } = renderHook(() => useActionState());

    await act(async () => {
      await result.current.execute(() =>
        Promise.reject(new Error("Network failed")),
      );
    });

    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBe("Network failed");
  });

  it("handles non-Error throws", async () => {
    const { result } = renderHook(() => useActionState());

    await act(async () => {
      await result.current.execute(() => Promise.reject("string error"));
    });

    expect(result.current.error).toBe("Unknown error");
  });

  it("resets state", async () => {
    const { result } = renderHook(() => useActionState());

    await act(async () => {
      await result.current.execute(() =>
        Promise.reject(new Error("fail")),
      );
    });

    expect(result.current.error).toBe("fail");

    act(() => {
      result.current.reset();
    });

    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("clears previous error on new execution", async () => {
    const { result } = renderHook(() => useActionState());

    await act(async () => {
      await result.current.execute(() =>
        Promise.reject(new Error("first")),
      );
    });

    expect(result.current.error).toBe("first");

    await act(async () => {
      await result.current.execute(() => Promise.resolve());
    });

    expect(result.current.error).toBeNull();
  });
});
