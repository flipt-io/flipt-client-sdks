using System;
using System.IO;
using System.Runtime.InteropServices;
using System.Text.Json;

namespace FliptClient
{
    internal static class NativeMethods
    {
        private static IntPtr _nativeLibraryHandle;

        [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
        public delegate IntPtr InitializeEngineDelegate(string opts);

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

        [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
        public delegate IntPtr GetSnapshotDelegate(IntPtr engine);

        public static InitializeEngineDelegate InitializeEngine;
        public static EvaluateVariantDelegate EvaluateVariant;
        public static EvaluateBooleanDelegate EvaluateBoolean;
        public static EvaluateBatchDelegate EvaluateBatch;
        public static ListFlagsDelegate ListFlags;
        public static DestroyEngineDelegate DestroyEngine;
        public static DestroyStringDelegate DestroyString;
        public static GetSnapshotDelegate GetSnapshot;

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
            GetSnapshot = GetDelegate<GetSnapshotDelegate>("get_snapshot");
        }

        private static string GetLibraryName()
        {
            string libraryName = GetPlatformLibraryPath();
            string baseDir = AppDomain.CurrentDomain.BaseDirectory;
            string directPath = Path.Combine(baseDir, libraryName);
            string runtimePath = Path.Combine(baseDir, "runtimes", libraryName);

            if (File.Exists(directPath))
            {
                return directPath;
            }

            if (File.Exists(runtimePath))
            {
                return runtimePath;
            }

            throw new FileNotFoundException($"Native library not found. Searched:\n{directPath}\n{runtimePath}");
        }

        private static string GetPlatformLibraryPath()
        {
            string libraryPath = string.Empty;

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

        private static T GetDelegate<T>(string functionName)
            where T : Delegate
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

        private static string GetString(IntPtr ptr)
        {
            if (ptr == IntPtr.Zero)
            {
                return string.Empty;
            }

            var str = Marshal.PtrToStringUTF8(ptr);
            DestroyString(ptr);
            return str ?? string.Empty;
        }
    }
}
