import { useState, useEffect } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRemixForm, getValidatedFormData } from "remix-hook-form";
import * as z from "zod";
import { Plus } from "lucide-react";
import { Button } from "~/components/ui/button";
import { Input } from "~/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "~/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "~/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "~/components/ui/table";
import type { Route } from "./+types/patients";
import { Form, useActionData, useNavigate, useSearchParams } from "react-router";
import { Label } from "~/components/ui/label";

// --- Interface (unchanged) ---
export interface IPatientData {
  patient_id: number;
  address: string;
  age: number;
  birthdate: string;
  email: string;
  firstname: string;
  gender: string;
  lastname: string;
  phonenumber: string;
  register: string;
}

export async function loader({ request }: Route.ClientLoaderArgs) {
  const url = new URL(request.url);
  const limit = url.searchParams.get("limit") || "10";
  const offset = url.searchParams.get("offset") || "0";

  const apiUrl = new URL("http://localhost:8080/patients");
  apiUrl.searchParams.set("limit", limit);
  apiUrl.searchParams.set("offset", offset);

  try {
    const res = await fetch(apiUrl.toString());
    if (!res.ok) {
      console.error("Өвчтөнүүдийг татахад алдаа гарлаа:", res.statusText);
      return [];
    }
    const data = await res.json();
    return data as IPatientData[];
  } catch (error) {
    console.error("Өвчтөнүүдийг татах үеийн алдаа:", error);
    return [];
  }
}

export async function action({ request }: Route.ClientActionArgs) {
  const { receivedValues, errors, data } =
    await getValidatedFormData<FormData>(request, resolver);

  if (errors) {
    return { errors, receivedValues };
  }

  try {
    const postData = data as Partial<IPatientData>;
    postData.age = calculateAge(data?.birthdate);

    const response = await fetch("http://localhost:8080/patients", {
      method: "POST",
      headers: {
        "Content-type": "application/json",
      },
      body: JSON.stringify(postData),
    });

    if (!response.ok) {
      const errorData = await response.text();
      console.error("API Алдаа:", errorData);
      return {
        apiError: `Өвчтөн нэмэхэд алдаа гарлаа: ${response.statusText}`,
        receivedValues,
      };
    }

    const newPatient = await response.json();
    return { success: true, patient: newPatient, revalidate: true };
  } catch (error) {
    console.error("Өвчтөн үүсгэх үеийн алдаа:", error);
    return { apiError: "Гэнэтийн алдаа гарлаа.", receivedValues };
  }
}

function calculateAge(birthdate: string): number {
  if (!birthdate) return 0;
  try {
    const today = new Date();
    const birthDate = new Date(birthdate);
    if (isNaN(birthDate.getTime())) return 0;

    let age = today.getFullYear() - birthDate.getFullYear();
    const m = today.getMonth() - birthDate.getMonth();
    if (m < 0 || (m === 0 && today.getDate() < birthDate.getDate())) {
      age--;
    }
    return age > 0 ? age : 0;
  } catch (e) {
    console.error("Нас тооцоолоход алдаа гарлаа:", e);
    return 0;
  }
}

const patientFormSchema = z.object({
  firstname: z
    .string()
    .min(2, { message: "Нэр дор хаяж 2 тэмдэгттэй байх ёстой." }),
  lastname: z
    .string()
    .min(2, { message: "Овог дор хаяж 2 тэмдэгттэй байх ёстой." }),
  email: z
    .string()
    .email({ message: "Зөв и-мэйл хаяг оруулна уу." }),
  phonenumber: z
    .string()
    .min(8, { message: "Утасны дугаар дор хаяж 8 оронтой байх ёстой." }),
  birthdate: z.string().refine((date) => !isNaN(Date.parse(date)), {
    message: "Зөв огноо оруулна уу.",
  }),
  register: z
    .string()
    .regex(/^[A-ZА-Я]{2}\d{8}$/u, {
      message: "Регистр нь 2 крилл/латин үсэг, 8 тооноос бүрдсэн байх ёстой.",
    }),
  gender: z.string().min(1, { message: "Хүйсээ сонгоно уу." }),
  address: z
    .string()
    .min(5, { message: "Хаяг дор хаяж 5 тэмдэгттэй байх ёстой." }),
});

type FormData = z.infer<typeof patientFormSchema>;
const resolver = zodResolver(patientFormSchema);

// --- Component ---
export default function PatientsPage({ loaderData }: Route.ComponentProps) {
  const navigate = useNavigate();
  const actionData = useActionData<typeof action>();
  const [searchParams, setSearchParams] = useSearchParams();

  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const { handleSubmit, setValue, register, reset, formState } =
    useRemixForm<FormData>({
      mode: "onSubmit",
      resolver,
    });

  useEffect(() => {
    if (actionData?.success && !actionData.errors) {
      setIsDialogOpen(false);
      reset();
      if (actionData.revalidate) {
        navigate(".", { replace: true, preventScrollReset: true });
      }
    }
  }, [actionData, reset, navigate]);

  const patientsOnPage = loaderData || []; // loaderData is IPatientData[] for the current page
  const limit = parseInt(searchParams.get("limit") || "10", 10);
  const offset = parseInt(searchParams.get("offset") || "0", 10);

  const currentPage = Math.floor(offset / limit) + 1;

  const hasNextPage = patientsOnPage.length === limit;
  const hasPreviousPage = offset > 0;

  const handlePageChange = (newPage: number) => {
    const newOffset = Math.max(0, (newPage - 1) * limit);
    const newSearchParams = new URLSearchParams(searchParams);
    newSearchParams.set("limit", String(limit));
    newSearchParams.set("offset", String(newOffset));
    setSearchParams(newSearchParams);
  };

  const handleRowsPerPageChange = (value: string) => {
    const newLimit = parseInt(value, 10);
    const newSearchParams = new URLSearchParams(searchParams);
    newSearchParams.set("limit", String(newLimit));
    newSearchParams.set("offset", "0");
    setSearchParams(newSearchParams);
  };

  const handleRowClick = (patientId: number) => {
    navigate(`/patients/${patientId}`);
  };

  const FormError = ({ name }: { name: keyof FormData }) => {
    const error = formState.errors[name]?.message;
    return error ? (
      <p className="text-sm text-red-500 mt-1">{String(error)}</p>
    ) : null;
  };

  const displayStartIndex = offset;
  const displayEndIndex = offset + patientsOnPage.length;

  return (
    <div className="container mx-auto py-8 px-4 md:px-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center mb-6 gap-4">
        <h1 className="text-2xl font-semibold">Өвчтөний удирдлага</h1>
        <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
          <DialogTrigger asChild>
            <Button className="flex items-center gap-2 w-full sm:w-auto">
              <Plus className="h-4 w-4" />
              Өвчтөн нэмэх
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-[600px]">
            <DialogHeader>
              <DialogTitle>Шинэ өвчтөн нэмэх</DialogTitle>
              <DialogDescription>
                Доорх өвчтөний мэдээллийг бөглөнө үү.
              </DialogDescription>
            </DialogHeader>
            {actionData?.apiError && (
              <p className="text-sm text-red-500 mb-4">
                {actionData.apiError}
              </p>
            )}
            <Form
              method="POST"
              onSubmit={handleSubmit}
              className="space-y-4"
            >
              {/* Form fields remain the same */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-1">
                  <Label htmlFor="firstname">Нэр</Label>
                  <Input
                    id="firstname"
                    placeholder="Бат"
                    {...register("firstname")}
                    aria-invalid={!!formState.errors.firstname}
                  />
                  <FormError name="firstname" />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="lastname">Овог</Label>
                  <Input
                    id="lastname"
                    placeholder="Дорж"
                    {...register("lastname")}
                    aria-invalid={!!formState.errors.lastname}
                  />
                  <FormError name="lastname" />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="email">И-мэйл</Label>
                  <Input
                    id="email"
                    type="email"
                    placeholder="bat.dorj@example.com"
                    {...register("email")}
                    aria-invalid={!!formState.errors.email}
                  />
                  <FormError name="email" />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="register">Регистрийн дугаар</Label>
                  <Input
                    id="register"
                    placeholder="УЕ98203192"
                    {...register("register")}
                    aria-invalid={!!formState.errors.register}
                  />
                  <FormError name="register" />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="phonenumber">Утасны дугаар</Label>
                  <Input
                    id="phonenumber"
                    placeholder="99123456"
                    {...register("phonenumber")}
                    aria-invalid={!!formState.errors.phonenumber}
                  />
                  <FormError name="phonenumber" />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="birthdate">Төрсөн огноо</Label>
                  <Input
                    id="birthdate"
                    type="date"
                    {...register("birthdate")}
                    aria-invalid={!!formState.errors.birthdate}
                  />
                  <FormError name="birthdate" />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="gender">Хүйс</Label>
                  <Select
                    onValueChange={(value) => setValue("gender", value)}
                    name={register("gender").name}
                  >
                    <SelectTrigger
                      id="gender"
                      aria-invalid={!!formState.errors.gender}
                    >
                      <SelectValue placeholder="Хүйс сонгоно уу" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="Male">Эрэгтэй</SelectItem>
                      <SelectItem value="Female">Эмэгтэй</SelectItem>
                      <SelectItem value="Other">Бусад</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormError name="gender" />
                </div>
                <div className="space-y-1 md:col-span-2">
                  <Label htmlFor="address">Хаяг</Label>
                  <Input
                    id="address"
                    placeholder="Гэрийн хаяг, Хот, Улс"
                    {...register("address")}
                    aria-invalid={!!formState.errors.address}
                  />
                  <FormError name="address" />
                </div>
              </div>

              <div className="flex justify-end pt-4">
                <Button type="submit" disabled={formState.isSubmitting}>
                  {formState.isSubmitting ? "Нэмж байна..." : "Өвчтөн нэмэх"}
                </Button>
              </div>
            </Form>
          </DialogContent>
        </Dialog>
      </div>

      <div className="rounded-md border shadow-sm overflow-x-auto mt-6">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[80px]">Дугаар</TableHead>
              <TableHead>Нэр</TableHead>
              <TableHead className="w-[80px]">Нас</TableHead>
              <TableHead>Хүйс</TableHead>
              <TableHead>И-мэйл</TableHead>
              <TableHead>Утас</TableHead>
              <TableHead>Регистр</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {patientsOnPage && patientsOnPage.length > 0 ? (
              patientsOnPage.map((patient) => ( // Map over the data for the current page
                <TableRow
                  key={patient.patient_id}
                  onClick={() => handleRowClick(patient.patient_id)}
                  className="cursor-pointer hover:bg-muted/50"
                >
                  <TableCell className="font-medium">
                    {patient.patient_id}
                  </TableCell>
                  <TableCell>{`${patient.firstname} ${patient.lastname}`}</TableCell>
                  <TableCell>{patient.age}</TableCell>
                  <TableCell>{patient.gender === "Male" ? "Эрэгтэй" : patient.gender === "Female" ? "Эмэгтэй" : "Бусад"}</TableCell>
                  <TableCell>{patient.email}</TableCell>
                  <TableCell>{patient.phonenumber}</TableCell>
                  <TableCell>{patient.register}</TableCell>
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={7} className="h-24 text-center">
                  {offset > 0 ? "Энэ хуудсанд өвчтөн олдсонгүй." : "Өвчтөн олдсонгүй."}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      {/* --- Pagination Controls (adapting to missing total count) --- */}
      {/* Show controls only if there are patients on the current page OR if we are not on the first page */}
      {(patientsOnPage.length > 0 || offset > 0) && (
        <div className="flex flex-col sm:flex-row items-center justify-between mt-6 gap-4">
          <div className="text-sm text-muted-foreground">
            {/* Show range for the current page */}
            {patientsOnPage.length > 0
              ? `${displayStartIndex + 1}-${displayEndIndex}-г харуулж байна`
              : "Үр дүн байхгүй"}
          </div>
          <div className="flex items-center gap-2">
            <span className="text-sm mr-2">
              Нэг хуудсанд харуулах мөрийн тоо:
            </span>
            <Select
              value={String(limit)} // Use limit from URL state
              onValueChange={handleRowsPerPageChange}
            >
              <SelectTrigger className="w-[70px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {[5, 10, 20, 50].map((size) => (
                  <SelectItem key={size} value={String(size)}>
                    {size}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {/* Display only the current page number */}
            <span className="text-sm mx-2">
              Хуудас {currentPage}
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={() => handlePageChange(currentPage - 1)}
              disabled={!hasPreviousPage} // Disable if offset is 0
            >
              Өмнөх
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => handlePageChange(currentPage + 1)}
              disabled={!hasNextPage} // Disable if last fetch returned less than limit items
            >
              Дараах
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

