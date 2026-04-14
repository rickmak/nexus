// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "Nexus",
    platforms: [
        .macOS(.v13)
    ],
    dependencies: [],
    targets: [
        .executableTarget(
            name: "Nexus",
            dependencies: [],
            path: "Nexus"
        )
    ]
)
