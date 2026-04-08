"use client";

import { Group, Button, Text } from '@mantine/core';
import { Slot } from '@/lib/api';
import dayjs from 'dayjs';
import customParseFormat from 'dayjs/plugin/customParseFormat';

// Extend dayjs with custom parse format
dayjs.extend(customParseFormat);

// Format time from HH:mm:ss or HH:mm:ss.ssssss or HH:mm to HH:mm
function formatTime(timeStr: string): string {
  // Remove microseconds if present
  const cleanStr = timeStr.split('.')[0];
  // Already in HH:mm format (HH:mm has 5 chars: "09:00")
  if (cleanStr.length === 5 && cleanStr.includes(':')) {
    return cleanStr;
  }
  // Has seconds - parse and format
  const parsed = dayjs(cleanStr, "HH:mm:ss");
  return parsed.isValid() ? parsed.format("HH:mm") : cleanStr;
}

interface SlotPickerProps {
  slots: Slot[];
  selectedSlot: Slot | null;
  onSelect: (slot: Slot) => void;
}

export default function SlotPicker({ slots, selectedSlot, onSelect }: SlotPickerProps) {
  if (slots.length === 0) {
    return <Text c="dimmed">Нет доступных слотов на эту дату</Text>;
  }

  return (
    <Group gap="xs" wrap="wrap">
      {slots.map((slot) => (
        <Button
          key={slot.id}
          variant={selectedSlot?.id === slot.id ? 'filled' : 'outline'}
          disabled={slot.isBooked}
          onClick={() => !slot.isBooked && onSelect(slot)}
          size="sm"
        >
          {formatTime(slot.startTime)}
        </Button>
      ))}
    </Group>
  );
}
