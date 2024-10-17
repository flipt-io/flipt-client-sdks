using System;
using System.Runtime.InteropServices;

namespace FliptClient
{
    internal static class NativeMethods
    {
        private const string DllName = "fliptengine";

        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr InitializeEngine(string @namespace, string opts);

        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr EvaluateVariant(IntPtr engine, string evaluationRequest);

        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr EvaluateBoolean(IntPtr engine, string evaluationRequest);

        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern void DestroyEngine(IntPtr engine);

        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern void DestroyString(IntPtr str);
    }
}