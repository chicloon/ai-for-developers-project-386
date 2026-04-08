"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Center, Loader } from "@mantine/core";

export default function AdminPage() {
  const router = useRouter();

  useEffect(() => {
    // Redirect to the new schedule management page
    router.push("/my/schedule");
  }, [router]);

  return (
    <Center h="100vh">
      <Loader />
    </Center>
  );
}
