import SwiftUI

@main
struct NexusApp: App {
    var body: some Scene {
        WindowGroup {
            ContentView()
                .preferredColorScheme(.dark)
                .frame(minWidth: 1100, minHeight: 700)
        }
        .defaultSize(width: 1100, height: 700)
    }
}
