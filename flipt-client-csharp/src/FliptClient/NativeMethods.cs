using System;
using System.Runtime.InteropServices;
using System.IO;

namespace FliptClient
{
    internal static class NativeMethods
    {
        private static IntPtr _nativeLibraryHandle;

        [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
        public delegate IntPtr InitializeEngineDelegate(string ns, string opts);

        [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
        public delegate IntPtr EvaluateVariantDelegate(IntPtr engine, string request);

        [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
        public delegate IntPtr EvaluateBooleanDelegate(IntPtr engine, string request);

        [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
        public delegate IntPtr EvaluateBatchDelegate(IntPtr engine, string request);

        [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
        public delegate IntPtr ListFlagsDelegate(IntPtr engine);

        [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
        public delegate void DestroyEngineDelegate(IntPtr engine);

        [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
        public delegate void DestroyStringDelegate(IntPtr str);

        public static InitializeEngineDelegate InitializeEngine;
        public static EvaluateVariantDelegate EvaluateVariant;
        public static EvaluateBooleanDelegate EvaluateBoolean;
        public static EvaluateBatchDelegate EvaluateBatch;
        public static ListFlagsDelegate ListFlags;
        public static DestroyEngineDelegate DestroyEngine;
        public static DestroyStringDelegate DestroyString;

        static NativeMethods()
        {
            string libraryPath = GetLibraryName();

            if (!File.Exists(libraryPath))
            {
                throw new FileNotFoundException($"Native library not found at path: {libraryPath}");
            }

            _nativeLibraryHandle = NativeLibrary.Load(libraryPath);

            if (_nativeLibraryHandle == IntPtr.Zero)
            {
                throw new InvalidOperationException("Failed to load native library, handle is null.");
            }

            // Initialize the delegates
            InitializeEngine = GetDelegate<InitializeEngineDelegate>("initialize_engine");
            EvaluateVariant = GetDelegate<EvaluateVariantDelegate>("evaluate_variant");
            EvaluateBoolean = GetDelegate<EvaluateBooleanDelegate>("evaluate_boolean");
            EvaluateBatch = GetDelegate<EvaluateBatchDelegate>("evaluate_batch");
            ListFlags = GetDelegate<ListFlagsDelegate>("list_flags");
            DestroyEngine = GetDelegate<DestroyEngineDelegate>("destroy_engine");
            DestroyString = GetDelegate<DestroyStringDelegate>("destroy_string");
        }

        private static string GetLibraryName()
        {
            string libraryName = GetPlatformLibraryPath();
            string libraryPath = Path.Combine(AppDomain.CurrentDomain.BaseDirectory, libraryName);

            if (!File.Exists(libraryPath))
            {
                // Try to find the library in the NuGet package structure
                libraryPath = Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "runtimes", libraryName);
            }

            return libraryPath;
        }

        private static string GetPlatformLibraryPath()
        {
            string libraryPath = "";

            if (RuntimeInformation.IsOSPlatform(OSPlatform.Windows))
            {
                libraryPath = "win-x64/native/fliptengine.dll";
            }
            else if (RuntimeInformation.IsOSPlatform(OSPlatform.Linux))
            {
                if (RuntimeInformation.ProcessArchitecture == Architecture.X64)
                {
                    libraryPath = "linux-x64/native/libfliptengine.so";
                }
                else if (RuntimeInformation.ProcessArchitecture == Architecture.Arm64)
                {
                    libraryPath = "linux-arm64/native/libfliptengine.so";
                }
            }
            else if (RuntimeInformation.IsOSPlatform(OSPlatform.OSX))
            {
                if (RuntimeInformation.ProcessArchitecture == Architecture.X64)
                {
                    libraryPath = "osx-x64/native/libfliptengine.dylib";
                }
                else if (RuntimeInformation.ProcessArchitecture == Architecture.Arm64)
                {
                    libraryPath = "osx-arm64/native/libfliptengine.dylib";
                }
            }

            return libraryPath;
        }

        private static T GetDelegate<T>(string functionName) where T : Delegate
        {
            if (_nativeLibraryHandle == IntPtr.Zero)
            {
                throw new InvalidOperationException("Native library not loaded.");
            }

            IntPtr procAddress = NativeLibrary.GetExport(_nativeLibraryHandle, functionName);
            if (procAddress == IntPtr.Zero)
            {
                throw new InvalidOperationException($"Function {functionName} not found.");
            }

            return Marshal.GetDelegateForFunctionPointer<T>(procAddress);
        }
    }
}
