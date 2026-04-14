# Nexus UI Comparison - Workspace Card

Three implementations of the same workspace card component for comparison.

## Running Apps

### 1. Wails + Svelte 5 (Web Preview)
```bash
cd nexus-wails/frontend
npm run dev
```
Open: http://localhost:5173/

### 2. Lynx (ReactLynx - Mobile/Web)
```bash
cd nexus-lynx
npm run dev
```
Open: http://localhost:3002/main.lynx.bundle?fullscreen=true

### 3. SwiftUI (Native macOS)
```bash
cd nexus-swiftui
swift build
swift run
```
Or open `Package.swift` in Xcode and run.

## Design Specifications

See `design-specs.md` for comprehensive design tokens from the designer subagent.

## Comparison Criteria

1. **Visual Polish** - How close to "Conductor-level" native Mac feel?
2. **Animation Smoothness** - 60fps? Responsive hover states?
3. **Dev Velocity** - Time to implement, hot reload quality
4. **Integration** - gRPC with Go daemon ease

## Projects

| Project | Framework | Status | URL |
|---------|-----------|--------|-----|
| nexus-wails | Wails + Svelte 5 | ✅ Running | http://localhost:5173/ |
| nexus-lynx | Lynx (ReactLynx) | ✅ Running | http://localhost:3002/main.lynx.bundle?fullscreen=true |
| nexus-swiftui | SwiftUI | ⏸️ Requires Xcode | Native macOS app |
