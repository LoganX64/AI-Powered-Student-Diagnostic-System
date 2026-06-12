import { createContext, useContext } from "react";

export type Role = "admin" | "coach";

export const RoleContext = createContext<Role>("admin");

export const useRole = () => useContext(RoleContext);
