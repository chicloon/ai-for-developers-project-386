import type { Metadata } from "next";
import "@mantine/core/styles.css";
import "@mantine/dates/styles.css";
import "@mantine/notifications/styles.css";
import "@mantine/schedule/styles.css";
import {
  MantineProvider,
  ColorSchemeScript,
  createTheme,
  MantineColorsTuple,
} from "@mantine/core";
import { Notifications } from "@mantine/notifications";
import { DatesProvider } from "@mantine/dates";
import { AuthProvider } from "@/components/auth/AuthProvider";
import dayjs from "dayjs";
import "dayjs/locale/ru";

export const metadata: Metadata = {
  title: "Call Booking",
  description: "Бронирование времени для звонков",
};

// Light theme primary colors - Winter Frost (icy blue tones)
const lightPrimaryColors: MantineColorsTuple = [
  "#F0F8FF", // Alice Blue
  "#E1F0FA",
  "#C2E3F5",
  "#A3D6F0",
  "#6BB6E8",
  "#4A9FD8",
  "#3A8BC4",
  "#2A77B0",
  "#1A639C",
  "#0A4F88",
];

// Dark theme primary colors - Winter Night (deep blue with cyan accents)
const darkPrimaryColors: MantineColorsTuple = [
  "#E8F4F8",
  "#B8E0EC",
  "#88CCF0",
  "#58B8F4",
  "#28A4F8",
  "#00BFFF", // Deep Sky Blue
  "#0099CC",
  "#007399",
  "#004D66",
  "#002633",
];

const theme = createTheme({
  primaryColor: "blue",
  colors: {
    blue: lightPrimaryColors,
    darkBlue: darkPrimaryColors,
  },
  primaryShade: { light: 5, dark: 5 },
  other: {
    backgroundColors: {
      light: "#F7FAFC", // Icy white
      dark: "#0D1B2A",  // Deep winter night
    },
  },
});

// Настройка русской локали для dayjs
dayjs.locale("ru");

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="ru" suppressHydrationWarning>
      <head><ColorSchemeScript /></head>
      <body>
        <MantineProvider theme={theme} defaultColorScheme="dark">
          <DatesProvider settings={{ locale: "ru" }}>
            <Notifications position="top-right" zIndex={10000} />
            <AuthProvider>{children}</AuthProvider>
          </DatesProvider>
        </MantineProvider>
      </body>
    </html>
  );
}
