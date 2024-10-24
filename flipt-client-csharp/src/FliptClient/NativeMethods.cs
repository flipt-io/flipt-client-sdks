using System;
using System.Runtime.InteropServices;
using System.IO;

namespace FliptClient
{
    internal static class NativeMethods
    {
        private static IntPtr _nativeLibraryHandle;

        static NativeMethods()
        {

            Console.WriteLine("NativeMethods constructor called.");

            string libraryPath = GetLibraryName();

            Console.WriteLine($"Loading library from {libraryPath}");

            if (!File.Exists(libraryPath))
            {
                throw new FileNotFoundException($"Native library not found at path: {libraryPath}");
            }

            _nativeLibraryHandle = NativeLibrary.Load(libraryPath);

            if (_nativeLibraryHandle == IntPtr.Zero)
            {
                throw new InvalidOperationException("Failed to load native library, handle is null.");
            }

        }

        private static string GetLibraryName()
        {
            string libraryPath = "";
            // write current os and architecture
            Console.WriteLine($"Current OS: {RuntimeInformation.OSDescription}");
            Console.WriteLine($"Current Architecture: {RuntimeInformation.ProcessArchitecture}");
            if (RuntimeInformation.IsOSPlatform(OSPlatform.Windows))
            {
                libraryPath = "runtimes/win-x64/native/fliptengine.dll";
            }
            else if (RuntimeInformation.IsOSPlatform(OSPlatform.Linux))
            {
                if (RuntimeInformation.ProcessArchitecture == Architecture.X64)
                {
                    libraryPath = "runtimes/linux-x64/native/libfliptengine.so";
                }
                else if (RuntimeInformation.ProcessArchitecture == Architecture.Arm64)
                {
                    libraryPath = "runtimes/linux-arm64/native/libfliptengine.so";
                }
            }
            else if (RuntimeInformation.IsOSPlatform(OSPlatform.OSX))
            {
                if (RuntimeInformation.ProcessArchitecture == Architecture.X64)
                {
                    libraryPath = "runtimes/osx-x64/native/libfliptengine.dylib";
                }
                else if (RuntimeInformation.ProcessArchitecture == Architecture.Arm64)
                {
                    libraryPath = "runtimes/osx-arm64/native/libfliptengine.dylib";
                }
            }

            Console.WriteLine($"Current Directory: {Environment.CurrentDirectory}");

            Console.WriteLine($"Loading library from {libraryPath}");
            return libraryPath;
        }

        public static void Initialize()
        {
            // Access a static member to ensure the static constructor is called
            var handle = _nativeLibraryHandle;
        }

        public static T GetDelegate<T>(string functionName) where T : Delegate
        {
            // Access a static member to ensure the static constructor is called
            var handle = _nativeLibraryHandle;

            if (handle == IntPtr.Zero)
            {
                throw new InvalidOperationException("Native library not loaded.");
            }

            IntPtr procAddress = NativeLibrary.GetExport(handle, functionName);
            if (procAddress == IntPtr.Zero)
            {
                throw new InvalidOperationException($"Function {functionName} not found.");
            }
            return Marshal.GetDelegateForFunctionPointer<T>(procAddress);
        }

        public static Func<string, string, IntPtr> InitializeEngine = GetDelegate<Func<string, string, IntPtr>>("InitializeEngine");
        public static Func<IntPtr, string, IntPtr> EvaluateVariant = GetDelegate<Func<IntPtr, string, IntPtr>>("EvaluateVariant");
        public static Func<IntPtr, string, IntPtr> EvaluateBoolean = GetDelegate<Func<IntPtr, string, IntPtr>>("EvaluateBoolean");
        public static Func<IntPtr, string, IntPtr> EvaluateBatch = GetDelegate<Func<IntPtr, string, IntPtr>>("EvaluateBatch");
        public static Func<IntPtr, IntPtr> ListFlags = GetDelegate<Func<IntPtr, IntPtr>>("ListFlags");
        public static Action<IntPtr> DestroyEngine = GetDelegate<Action<IntPtr>>("DestroyEngine");
        public static Action<IntPtr> DestroyString = GetDelegate<Action<IntPtr>>("DestroyString");
    }
}
