import "@testing-library/jest-dom/vitest"
import { vi } from "vitest"

Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => undefined,
    removeListener: () => undefined,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    dispatchEvent: () => false,
  }),
})

class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}

vi.stubGlobal("ResizeObserver", ResizeObserverStub)

vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue({
  width: 800,
  height: 256,
  top: 0,
  right: 800,
  bottom: 256,
  left: 0,
  x: 0,
  y: 0,
  toJSON: () => ({}),
})

Object.defineProperties(Range.prototype, {
  getBoundingClientRect: { value: () => ({
    width: 0, height: 0, top: 0, right: 0, bottom: 0, left: 0, x: 0, y: 0,
    toJSON: () => ({}),
  }) },
  getClientRects: { value: () => [] },
})

Object.defineProperty(Element.prototype, "getAnimations", {
  value: () => [],
})
