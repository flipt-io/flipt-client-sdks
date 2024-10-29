// swift-tools-version: 5.9.2

import PackageDescription

let package = Package(
    name: "FliptClient",
    platforms: [.iOS(.v13), .macOS(.v10_15)],
    products: [
        .library(
            name: "FliptClient",
            targets: ["FliptClient"]),
    ],
    targets: [
        .target(
            name: "FliptClient",
            dependencies: ["FliptEngineFFI"],
            linkerSettings: [
              .linkedFramework("SystemConfiguration"),
              .linkedLibrary("fliptengine")
            ]
        ),
        .binaryTarget(
            name: "FliptEngineFFI",
            path: "./Sources/FliptEngineFFI.xcframework"),
        .testTarget(
            name: "FliptClientTests",
            dependencies: ["FliptClient"]),
    ]
)
