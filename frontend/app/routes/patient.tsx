import { useLoaderData, Link, useRevalidator } from "react-router";
import { ArrowLeft, BrainCircuit, CalendarDays, FileText, Microscope, Pill, Stethoscope } from "lucide-react"; // Added icons
import { Button } from "~/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "~/components/ui/card";
import { Separator } from "~/components/ui/separator";
import { useState } from "react"; // Import useState
import { SymptomPredictionDialog } from "~/components/symptom-prediction-dialog";
import type { Route } from "./+types/patient";
import { Badge } from "~/components/ui/badge";
import { decodeBase64Utf8 } from "~/lib/utils";

export interface IPatientData {
  patient_id: number;
  address: string | null; // Allow null from DB
  age: number;
  birthdate: string | null; // Allow null from DB
  email: string;
  firstname: string;
  gender: string;
  lastname: string;
  phonenumber: string;
  register: string;
}

export interface IDiseaseTreatment {
  treatment: string[];
}

export interface IFullDiseaseData {
  disease_id: number;
  disease_name: string;
  disease_code: string;
  disease_description: string | null;
  disease_treatment: IDiseaseTreatment | null; // Parsed JSONB
  created_at: string | null; // Assuming string timestamp
  updated_at: string | null; // Assuming string timestamp
}

interface LoaderData {
  patient: IPatientData;
  history: IEnrichedDiseaseInstance[];
}

export interface IPatientDiseaseInstance {
  patient_disease_id: number;
  patient_id: number;
  disease_id: number;
  disease_name: string; // From JOIN
  disease_code: string; // From JOIN
  diagnosis_date: string | null; // Assuming string date YYYY-MM-DD
  notes: string | null;
  created_at: string | null; // Assuming string timestamp
  updated_at: string | null; // Assuming string timestamp
  // Treatment might NOT be directly here, hence fetching IFullDiseaseData
}

export interface ILinkedSymptom {
  symptom_id: number;
  symptom_name: string;
  symptom_description: string | null;
}

export interface IEnrichedDiseaseInstance extends IPatientDiseaseInstance {
  linkedSymptoms: ILinkedSymptom[];
  disease_treatment: IDiseaseTreatment | null; // Added from full disease fetch
}

export async function loader({ params }: Route.LoaderArgs) {
  const patientId = params.patientId;
  if (!patientId) {
    throw new Response("Patient ID is required", { status: 400 });
  }

  const patientUrl = `http://localhost:8080/patients/${patientId}`;
  const instancesUrl = `http://localhost:8080/patients/${patientId}/disease-instances`;

  try {
    const [patientRes, instancesRes] = await Promise.all([
      fetch(patientUrl),
      fetch(instancesUrl),
    ]);

    if (!patientRes.ok) {
      if (patientRes.status === 404) throw new Response(`Patient not found (ID: ${patientId})`, { status: 404 });
      throw new Response(`Failed to fetch patient data (${patientRes.status})`, { status: patientRes.status });
    }
    const patient: IPatientData = await patientRes.json();

    let instances: IPatientDiseaseInstance[] = [];
    if (instancesRes.ok) {
      instances = await instancesRes.json();
    } else {
      // Log warning but continue, patient might just have no history
      console.warn(`Failed to fetch disease instances for patient ${patientId} (${instancesRes.status}), proceeding with empty history.`);
    }

    if (instances.length === 0) {
        return { patient, history: [] };
    }

    const symptomFetchPromises = instances.map(instance => {
      const symptomsUrl = `http://localhost:8080/disease-instances/${instance.patient_disease_id}/symptoms`;
      return fetch(symptomsUrl).then(async (res) => { // Make async
        if (!res.ok) {
          console.error(`Failed to fetch symptoms for instance ${instance.patient_disease_id} (${res.status})`);
          return []; // Return empty array on error for this instance
        }
        try {
            const data = await res.json();
            console.log(data)
            return data as ILinkedSymptom[];
        } catch (e) {
            console.error(`Failed to parse symptoms JSON for instance ${instance.patient_disease_id}:`, e);
            return []; // Return empty on parse error
        }
      });
    });

    const linkedSymptomsArrays = await Promise.all(symptomFetchPromises);

    const uniqueDiseaseIds = [...new Set(instances.map(inst => inst.disease_id))];
    const diseaseDetailPromises = uniqueDiseaseIds.map(async id => {
        const diseaseUrl = `http://localhost:8080/diseases/${id}`;
        return fetch(diseaseUrl).then(async (res) => { // Make async
            if (!res.ok) {
                console.error(`Failed to fetch disease details for ID ${id} (${res.status})`);
                return null; // Handle error gracefully
            }
             try {
                const data = await res.json();
                // Ensure treatment is parsed correctly if it's a stringified JSON
                console.log(data)
                const decodedJsonString = decodeBase64Utf8(data.disease_treatment);
                if (decodedJsonString) {
                   try {
                       data.disease_treatment = JSON.parse(decodedJsonString);
                   } catch (parseError) {
                       console.error(`Failed to parse disease_treatment JSON for disease ${id}:`, parseError);
                       data.disease_treatment = null;
                   }
                }
                return data as IFullDiseaseData;
            } catch (e) {
                console.error(`Failed to parse disease JSON for ID ${id}:`, e);
                return null; // Return null on parse error
            }
        });
    });
    const diseaseDetailsResults = await Promise.all(diseaseDetailPromises);
    const diseaseDetailsMap = new Map<number, IFullDiseaseData>();
    diseaseDetailsResults.forEach(detail => {
        if (detail) {
            diseaseDetailsMap.set(detail.disease_id, detail);
        }
    });


    // 4. Combine instance data with linked symptoms and treatment info
    const history: IEnrichedDiseaseInstance[] = instances.map((instance, index) => {
      const fullDisease = diseaseDetailsMap.get(instance.disease_id);
      return {
        ...instance, // Spread properties from the instance fetch
        // Use treatment from the detailed fetch, default to null if not found
        disease_treatment: fullDisease?.disease_treatment ?? null,
        linkedSymptoms: linkedSymptomsArrays[index] || [], // Add the fetched symptoms
      };
    });

    // Sort history by diagnosis date descending (most recent first)
    // Handle null dates by putting them last or first based on preference
    history.sort((a, b) => {
        if (!a.diagnosis_date && !b.diagnosis_date) return 0;
        if (!a.diagnosis_date) return 1; // Put nulls last
        if (!b.diagnosis_date) return -1; // Put nulls last
        return new Date(b.diagnosis_date).getTime() - new Date(a.diagnosis_date).getTime();
    });


    return { patient, history };

  } catch (error) {
    console.error("Error in patient detail loader:", error);
    if (error instanceof Response) throw error; // Re-throw Response errors (like 404)
    throw new Response("Could not load patient history data", { status: 500 });
  }
}

// --- Component ---
export default function PatientDetailPage({params}: Route.ComponentProps) {
  const { patient, history } = useLoaderData<LoaderData>();
  const [isPredictionDialogOpen, setIsPredictionDialogOpen] = useState(false);
  const revalidator = useRevalidator(); // Get the revalidator function

  const formatDate = (dateString: string | null | undefined) => {
    if (!dateString) return "Тодорхойгүй";
    try {
      return new Date(dateString).toLocaleDateString();
    } catch (e) {
      console.warn("Error formatting date:", dateString, e);
      return dateString; // Return original string if formatting fails
    }
  };

  return (
    <div className="container mx-auto py-8 px-4 md:px-6 lg:px-8 space-y-6">
      <div className="flex justify-between items-center flex-wrap gap-2">
        <Button variant="outline" size="sm" asChild>
          <Link to="/patients" className="flex items-center gap-2">
            <ArrowLeft className="h-4 w-4" />
            Буцах
          </Link>
        </Button>
        <h1 className="text-xl md:text-2xl font-semibold text-center flex-grow truncate px-4">
          {patient.firstname} {patient.lastname} ({patient.register})
        </h1>
        <Button onClick={() => setIsPredictionDialogOpen(true)} variant="default">
          <BrainCircuit className="mr-2 h-4 w-4" />
          Таамаглал / Онош Нэмэх
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Үндсэн мэдээлэл</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-x-6 gap-y-3 text-sm">
            <div><span className="font-semibold text-muted-foreground">Нэр: </span>{patient.firstname} {patient.lastname}</div>
            <div><span className="font-semibold text-muted-foreground">Нас: </span>{patient.age}</div>
            <div><span className="font-semibold text-muted-foreground">Хүйс: </span>{patient.gender === "Male" ? "Эрэгтэй" : patient.gender === "Female" ? "Эмэгтэй" : patient.gender || "Тодорхойгүй"}</div>
            <div><span className="font-semibold text-muted-foreground">Төрсөн огноо: </span>{formatDate(patient.birthdate)}</div>
            <div><span className="font-semibold text-muted-foreground">И-мэйл: </span>{patient.email}</div>
            <div><span className="font-semibold text-muted-foreground">Утас: </span>{patient.phonenumber}</div>
            <div><span className="font-semibold text-muted-foreground">Регистр: </span>{patient.register}</div>
            <div className="sm:col-span-2 lg:col-span-3"><span className="font-semibold text-muted-foreground">Хаяг: </span>{patient.address || "Бүртгэлгүй"}</div>
          </div>
        </CardContent>
      </Card>

      <Separator />

      {/* Medical History Section */}
      <div>
        <h2 className="text-xl font-semibold mb-4 flex items-center gap-2">
          <Stethoscope className="h-5 w-5" />
          Эмчилгээний түүх
        </h2>
        {history && history.length > 0 ? (
          <div className="space-y-4">
            {history.map((instance) => (
              <Card key={instance.patient_disease_id} className="border-l-4 border-blue-500">
                <CardHeader className="pb-3">
                  <CardTitle className="text-lg">{instance.disease_name} {instance.disease_code && `(${instance.disease_code})`}</CardTitle>
                  <CardDescription className="flex items-center gap-2 text-xs text-muted-foreground pt-1">
                    <CalendarDays className="h-3 w-3" />
                    Оношлогдсон огноо: {formatDate(instance.created_at)}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-3 text-sm">
                  {/* Notes */}
                  {instance.notes && (
                    <div className="flex items-start gap-2">
                      <FileText className="h-4 w-4 mt-0.5 text-muted-foreground flex-shrink-0" />
                      <p><span className="font-semibold">Тэмдэглэл:</span> {instance.notes}</p>
                    </div>
                  )}

                  {/* Linked Symptoms */}
                  {instance.linkedSymptoms && instance.linkedSymptoms.length > 0 && (
                    <div className="flex items-start gap-2">
                       <Microscope className="h-4 w-4 mt-0.5 text-muted-foreground flex-shrink-0" />
                       <div>
                         <span className="font-semibold">Илэрсэн шинж тэмдэг:</span>
                         <div className="flex flex-wrap gap-1 mt-1">
                           {instance.linkedSymptoms.map(symptom => (
                             <Badge key={symptom.symptom_id} variant="outline">{symptom.symptom_name}</Badge>
                           ))}
                         </div>
                       </div>
                    </div>
                  )}

                  {/* Treatments */}
                  {instance.disease_treatment?.treatment && instance.disease_treatment.treatment.length > 0 && (
                     <div className="flex items-start gap-2">
                       <Pill className="h-4 w-4 mt-0.5 text-muted-foreground flex-shrink-0" />
                       <div>
                         <span className="font-semibold">Санал болгосон эмчилгээ:</span>
                         <div className="flex flex-wrap gap-1 mt-1">
                           {instance.disease_treatment.treatment.map((treatment, index) => (
                             <Badge key={index} variant="secondary">{treatment}</Badge>
                           ))}
                         </div>
                       </div>
                    </div>
                  )}

                   {/* Show if no symptoms/treatments were linked/found */}
                   {(!instance.linkedSymptoms || instance.linkedSymptoms.length === 0) &&
                    (!instance.disease_treatment?.treatment || instance.disease_treatment.treatment.length === 0) &&
                    !instance.notes && (
                       <p className="text-xs text-muted-foreground italic">Энэ оноштой холбоотой нэмэлт мэдээлэл (шинж тэмдэг, эмчилгээ, тэмдэглэл) бүртгэгдээгүй байна.</p>
                   )}

                </CardContent>
              </Card>
            ))}
          </div>
        ) : (
          <p className="text-muted-foreground text-center py-4">
            Бүртгэгдсэн эмчилгээний түүх олдсонгүй.
          </p>
        )}
      </div>

      {/* Render the Dialog Component */}
      <SymptomPredictionDialog
        open={isPredictionDialogOpen}
        onOpenChange={setIsPredictionDialogOpen}
        patientId={patient?.patient_id}
        onSaveSuccess={() => {
            console.log("Save successful, revalidating...");
            revalidator.revalidate();
        }}
      />
    </div>
  );
}
