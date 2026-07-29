import { View } from "react-native";
import { Text } from "@/components/ui/text";

/** Honest empty state while agent management remains a web/desktop action. */
export function NoAgentBanner() {
  return (
    <View className="mx-3 mt-2 mb-1 rounded-xl border border-border bg-secondary/50 px-3 py-2">
      <Text className="text-sm font-medium text-foreground">
        No agents available
      </Text>
      <Text className="text-xs text-muted-foreground mt-0.5">
        Add or enable an agent from the web or desktop app to start chatting.
      </Text>
    </View>
  );
}
