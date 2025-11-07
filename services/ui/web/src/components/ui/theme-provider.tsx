import { ThemeProvider as NextThemeProvider } from "next-themes";
import { type PropsWithChildren } from "react";

type ThemeProviderProps = PropsWithChildren<{
  attribute?: string;
  defaultTheme?: string;
  enableSystem?: boolean;
  disableTransitionOnChange?: boolean;
}>;

export function ThemeProvider({ children, ...props }: ThemeProviderProps) {
  return <NextThemeProvider {...props}>{children}</NextThemeProvider>;
}
