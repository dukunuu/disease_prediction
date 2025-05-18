import { useState, useEffect, useMemo } from "react";
import { Button } from "~/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "~/components/ui/dialog";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "~/components/ui/command";
import { Badge } from "~/components/ui/badge";
import { X as RemoveIcon, Loader2, AlertCircle, Check } from "lucide-react"; // Added Check icon
import { ScrollArea } from "~/components/ui/scroll-area";
import { cn } from "~/lib/utils"; // Assuming you have a utility for class names

// --- Interfaces (mostly the same, added PatientDisease response) ---
interface ISymptomOption {
  symptom_id: number;
  symptom_name: string;
  symptom_description?: string;
}

interface IPredictionRequest {
  known_symptoms: Record<string, number>;
}

interface IPredictionResponse {
  predictions: { disease: string; probability: string }[];
}

interface IDiseaseOption {
  disease_id: number;
  disease_name: string;
  disease_code?: string;
  disease_description?: string;
}

// Interface for the request to create a disease instance
interface IRecordPatientDiseaseInstanceRequest {
  disease_id: number;
  diagnosis_date?: string | null; // Optional: YYYY-MM-DD or null
  notes?: string | null;          // Optional
}

// Interface for the response when creating a disease instance
interface IPatientDiseaseInstanceResponse {
    patient_disease_id: number;
    patient_id: number;
    disease_id: number;
    diagnosis_date: string | null; // Assuming string date YYYY-MM-DD
    notes: string | null;
    created_at: string | null; // Assuming string timestamp
    updated_at: string | null; // Assuming string timestamp
}


// Interface for the request to link a symptom
interface ILinkSymptomRequest {
  symptom_id: number;
}

interface SymptomPredictionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  patientId: number | undefined;
  onSaveSuccess?: () => void; // Optional callback on successful save
}

export function SymptomPredictionDialog({
  open,
  onOpenChange,
  patientId,
  onSaveSuccess,
}: SymptomPredictionDialogProps) {
  // --- State variables ---
  const [allSymptoms, setAllSymptoms] = useState<ISymptomOption[]>([]);
  const [symptomsLoading, setSymptomsLoading] = useState(false);
  const [symptomsError, setSymptomsError] = useState<string | null>(null);
  const [selectedSymptoms, setSelectedSymptoms] = useState<ISymptomOption[]>([]);
  const [searchTerm, setSearchTerm] = useState("");
  const [predictionLoading, setPredictionLoading] = useState(false);
  const [predictions, setPredictions] = useState<IPredictionResponse['predictions'] | null>(null);
  const [predictionError, setPredictionError] = useState<string | null>(null);
  const [allDiseases, setAllDiseases] = useState<IDiseaseOption[]>([]);
  const [diseasesLoading, setDiseasesLoading] = useState(false);
  const [diseasesError, setDiseasesError] = useState<string | null>(null);
  // --- State Change: Store only ONE selected disease ---
  const [selectedDisease, setSelectedDisease] = useState<IDiseaseOption | null>(null);
  const [diseaseSearchTerm, setDiseaseSearchTerm] = useState("");
  const [saveLoading, setSaveLoading] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  // --- useEffect for fetching data (remains the same) ---
  useEffect(() => {
    if (open) {
      const fetchData = async () => {
        setSymptomsLoading(true);
        setDiseasesLoading(true);
        setSymptomsError(null);
        setDiseasesError(null);
        setAllSymptoms([]);
        setAllDiseases([]);
        setSelectedSymptoms([]);
        // --- Reset single selected disease ---
        setSelectedDisease(null);
        setPredictions(null);
        setPredictionError(null);
        setSaveError(null);

        try {
          const [symptomsResponse, diseasesResponse] = await Promise.all([
            fetch("http://localhost:8080/symptoms"),
            fetch("http://localhost:8080/diseases"),
          ]);

          if (!symptomsResponse.ok) throw new Error(`Шинж тэмдгүүдийг татахад алдаа гарлаа (${symptomsResponse.status})`);
          const symptomsData: ISymptomOption[] = await symptomsResponse.json();
          setAllSymptoms(symptomsData);

          if (!diseasesResponse.ok) throw new Error(`Өвчнүүдийг татахад алдаа гарлаа (${diseasesResponse.status})`);
          const diseasesData: IDiseaseOption[] = await diseasesResponse.json();
          setAllDiseases(diseasesData);

        } catch (error) {
          console.error("Error fetching initial data:", error);
          const errorMsg = error instanceof Error ? error.message : "Тодорхойгүй алдаа";
          setSymptomsError(errorMsg);
          setDiseasesError(errorMsg);
        } finally {
          setSymptomsLoading(false);
          setDiseasesLoading(false);
        }
      };
      fetchData();
    } else {
      setSearchTerm("");
      setDiseaseSearchTerm("");
    }
  }, [open]);

  // --- Symptom Selection Handlers (remain the same) ---
  const handleSelectSymptom = (symptom: ISymptomOption) => {
    if (!selectedSymptoms.some((s) => s.symptom_id === symptom.symptom_id)) {
      setSelectedSymptoms((prev) => [...prev, symptom]);
    }
    setSearchTerm("");
  };
  const handleRemoveSymptom = (symptomId: number) => {
    setSelectedSymptoms((prev) => prev.filter((s) => s.symptom_id !== symptomId));
  };

  // --- Disease Selection Handlers (Updated for single selection) ---
  const handleSelectDisease = (disease: IDiseaseOption) => {
    setSelectedDisease(disease); // Set the single selected disease
    setDiseaseSearchTerm("");
  };
  const handleRemoveDisease = () => { // No ID needed, just clear
    setSelectedDisease(null);
  };

  // --- Prediction Handler (remains the same) ---
  const handlePredict = async () => {
     if (selectedSymptoms.length === 0) {
      setPredictionError("Таамаглал хийхийн тулд дор хаяж нэг шинж тэмдэг сонгоно уу.");
      return;
    }
    setPredictionLoading(true);
    setPredictionError(null);
    setPredictions(null);
    setSaveError(null);
    // --- Reset single selected disease on new prediction ---
    setSelectedDisease(null);

    const known_symptoms_obj = selectedSymptoms.reduce(
      (accumulator, currentSymptom) => {
        accumulator[currentSymptom.symptom_name] = 1;
        return accumulator;
      },
      {} as Record<string, number> // Initial value is an empty object, typed correctly
    );
    const requestBody: IPredictionRequest = {
      known_symptoms: known_symptoms_obj,
    };

    try {
      const response = await fetch("http://localhost:8080/predict", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(requestBody),
      });
      if (!response.ok) {
        let errorMsg = `Таамаглал хийхэд алдаа гарлаа (${response.status})`;
        try { const errorData = await response.json(); errorMsg = errorData.message || errorData.detail || errorMsg; } catch (_) {}
        throw new Error(errorMsg);
      }
      const data: IPredictionResponse = await response.json();
      console.log(data)
      setPredictions(data.predictions);
    } catch (error) {
      console.error("Prediction error:", error);
      setPredictionError(error instanceof Error ? error.message : "Тодорхойгүй алдаа");
    } finally {
      setPredictionLoading(false);
    }
  };

  // --- *** REVISED Save Handler (Single Disease Instance + Symptom Linking) *** ---
  const handleSave = async () => {
    if (!patientId) {
      setSaveError("Өвчтөний дугаар тодорхойгүй байна.");
      return;
    }
    if (selectedSymptoms.length === 0) {
      setSaveError("Хадгалахын тулд дор хаяж нэг шинж тэмдэг сонгосон байх шаардлагатай.");
      return;
    }
    // --- Check single selected disease ---
    if (!selectedDisease) {
      setSaveError("Хадгалахын тулд нэг өвчин сонгосон байх шаардлагатай.");
      return;
    }

    setSaveLoading(true);
    setSaveError(null);

    let createdInstanceId: number | null = null;

    try {
      // --- Step 1: Create the Disease Instance ---
      const instanceUrl = `http://localhost:8080/patients/${patientId}/disease-instances`;
      const instanceBody: IRecordPatientDiseaseInstanceRequest = {
        disease_id: selectedDisease.disease_id,
        // Add diagnosis_date or notes here if needed, e.g.:
        // diagnosis_date: new Date().toISOString().split('T')[0], // Today's date
      };

      const instanceResponse = await fetch(instanceUrl, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(instanceBody),
      });

      if (!instanceResponse.ok) {
        let errorDetail = `Status: ${instanceResponse.status}`;
        try { errorDetail = await instanceResponse.text(); } catch (_) {}
        throw new Error(`Өвчний онош '${selectedDisease.disease_name}' бүртгэж чадсангүй: ${errorDetail}`);
      }

      const createdInstance: IPatientDiseaseInstanceResponse = await instanceResponse.json();
      createdInstanceId = createdInstance.patient_disease_id;
      console.log(`Disease instance created with ID: ${createdInstanceId}`);

      // --- Step 2: Link Selected Symptoms to the Created Instance ---
      if (createdInstanceId) {
        const symptomLinkPromises = selectedSymptoms.map(symptom => {
          const linkUrl = `http://localhost:8080/disease-instances/${createdInstanceId}/symptoms`;
          const linkBody: ILinkSymptomRequest = { symptom_id: symptom.symptom_id };
          return fetch(linkUrl, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(linkBody),
          }).then(async (res) => {
            if (!res.ok) {
              let errorDetail = `Status: ${res.status}`;
              try { errorDetail = await res.text(); } catch (_) {}
              // Throw specific error for linking failure
              throw new Error(`Шинж тэмдэг '${symptom.symptom_name}' холбож чадсангүй: ${errorDetail}`);
            }
            return { success: true, symptomName: symptom.symptom_name };
          });
        });

        // Wait for all symptom linking requests to settle
        const linkResults = await Promise.allSettled(symptomLinkPromises);
        const linkFailures = linkResults.filter(r => r.status === 'rejected');

        if (linkFailures.length > 0) {
          // Report which symptoms failed to link
          const errorMessages = linkFailures.map(f => (f as PromiseRejectedResult).reason?.message || 'Тодорхойгүй холболтын алдаа');
          throw new Error(`Онош бүртгэгдсэн (${selectedDisease.disease_name}), гэвч дараах шинж тэмдгүүдийг холбоход алдаа гарлаа: ${errorMessages.join("; ")}`);
        }

        console.log("All selected symptoms linked successfully to instance", createdInstanceId);
      }

      // All steps succeeded
      console.log("Disease Instance and Symptom Links saved successfully!");
      onSaveSuccess?.(); // Call the success callback
      onOpenChange(false); // Close dialog

    } catch (error) {
      console.error("Save error:", error);
      // Display the combined or specific error message
      setSaveError(error instanceof Error ? error.message : "Хадгалахад тодорхойгүй алдаа гарлаа.");
      // Note: If instance creation succeeded but linking failed, the instance still exists.
      // Consider adding logic to delete the instance if linking fails completely, if desired.
    } finally {
      setSaveLoading(false);
    }
  };

  // --- Memoized lists ---
  const availableSymptoms = useMemo(() => {
    const selectedIds = new Set(selectedSymptoms.map((s) => s.symptom_id));
    return allSymptoms.filter((symptom) => !selectedIds.has(symptom.symptom_id));
  }, [allSymptoms, selectedSymptoms]);

  // --- availableDiseases doesn't need filtering based on selection anymore ---
  // const availableDiseases = useMemo(() => {
  //   return allDiseases; // Just return all fetched diseases
  // }, [allDiseases]);

  // --- JSX ---
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[700px] max-h-[90vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>Шинж тэмдэг ба Өвчний Бүртгэл</DialogTitle>
          <DialogDescription>
            Өвчтөнд илэрсэн шинж тэмдгийг сонгож таамаглал хийн, оношийг баталгаажуулна уу.
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="flex-grow pr-6 -mr-6">
          {/* 1. Symptom Selection (Unchanged) */}
          <div className="space-y-4 py-2">
            <h4 className="font-medium text-sm mb-2">1. Шинж тэмдэг сонгох</h4>
            <Command shouldFilter={false} className="overflow-visible">
              <CommandInput placeholder="Шинж тэмдэг хайх..." value={searchTerm} onValueChange={setSearchTerm} disabled={symptomsLoading || diseasesLoading}/>
              <div className="mt-2 flex flex-wrap gap-1 min-h-[24px]">
                {selectedSymptoms.map((symptom) => (
                  <Badge key={symptom.symptom_id} variant="secondary">
                    {symptom.symptom_name}
                    <button onClick={() => handleRemoveSymptom(symptom.symptom_id)} className="ml-1 rounded-full outline-none ring-offset-background focus:ring-2 focus:ring-ring focus:ring-offset-2" aria-label={`Устгах ${symptom.symptom_name}`}>
                      <RemoveIcon className="h-3 w-3 text-muted-foreground hover:text-foreground" />
                    </button>
                  </Badge>
                ))}
              </div>
              <CommandList>
                {symptomsLoading && <div className="p-4 text-center text-sm text-muted-foreground flex items-center justify-center"><Loader2 className="mr-2 h-4 w-4 animate-spin" /> Ачааллаж байна...</div>}
                {symptomsError && <div className="p-4 text-center text-sm text-destructive flex items-center justify-center"><AlertCircle className="mr-2 h-4 w-4" /> {symptomsError}</div>}
                {!symptomsLoading && !symptomsError && (
                  <>
                    <CommandEmpty>{allSymptoms.length > 0 ? "Шинж тэмдэг олдсонгүй." : "Шинж тэмдгийн жагсаалт хоосон байна."}</CommandEmpty>
                    <ScrollArea className="max-h-[150px]">
                      <CommandGroup heading="Боломжит шинж тэмдгүүд">
                        {availableSymptoms
                          .filter(s => s.symptom_name.toLowerCase().includes(searchTerm.toLowerCase()))
                          .map((symptom) => (
                            <CommandItem key={symptom.symptom_id} value={symptom.symptom_name} onSelect={() => handleSelectSymptom(symptom)} className="cursor-pointer">
                              {symptom.symptom_name}
                            </CommandItem>
                          ))}
                      </CommandGroup>
                    </ScrollArea>
                  </>
                )}
              </CommandList>
            </Command>
            <Button size="sm" onClick={handlePredict} disabled={predictionLoading || selectedSymptoms.length === 0 || symptomsLoading || diseasesLoading} className="w-full">
              {predictionLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              2. Таамаглал хийх
            </Button>
          </div>

          {/* 3. Prediction Results (Unchanged) */}
          {(predictionLoading || predictionError || predictions) && (
             <div className="mt-4 space-y-2 py-2 border-t">
               <h4 className="font-medium text-sm mb-2">3. Таамаглалын үр дүн</h4>
               {predictionLoading && !predictions && <div className="p-4 text-center text-sm text-muted-foreground flex items-center justify-center"><Loader2 className="mr-2 h-4 w-4 animate-spin" /> Таамаглаж байна...</div>}
               {predictionError && (
                 <div className="text-sm text-destructive bg-destructive/10 p-3 rounded-md flex items-center">
                   <AlertCircle className="mr-2 h-4 w-4 flex-shrink-0" />
                   {predictionError}
                 </div>
               )}
               {predictions && !predictionError && (
                 <ScrollArea className="max-h-[100px]">
                   <ul className="list-disc list-inside text-sm bg-muted/50 p-3 rounded-md">
                     {Array.isArray(predictions) && predictions.length > 0 ? predictions.map((pred, index) => (
                       <li key={index}>
                         {`${pred.disease} (${pred.probability})`}
                       </li>
                     )) : <li>Таамаглал олдсонгүй.</li>}
                   </ul>
                 </ScrollArea>
               )}
            </div>
          )}

          {/* 4. Disease Selection (Updated for single selection) */}
          {predictions && !predictionError && (
            <div className="space-y-4 py-2 border-t">
              <h4 className="font-medium text-sm mb-2">4. Онош сонгох (Нэгийг сонгоно уу)</h4>
              <Command shouldFilter={false} className="overflow-visible">
                <CommandInput placeholder="Өвчин хайх..." value={diseaseSearchTerm} onValueChange={setDiseaseSearchTerm} disabled={diseasesLoading || symptomsLoading}/>
                {/* Display single selected disease */}
                <div className="mt-2 flex flex-wrap gap-1 min-h-[24px]">
                  {selectedDisease && (
                    <Badge variant="secondary">
                      {selectedDisease.disease_name}
                      <button onClick={handleRemoveDisease} className="ml-1 rounded-full outline-none ring-offset-background focus:ring-2 focus:ring-ring focus:ring-offset-2" aria-label={`Устгах ${selectedDisease.disease_name}`}>
                        <RemoveIcon className="h-3 w-3 text-muted-foreground hover:text-foreground" />
                      </button>
                    </Badge>
                  )}
                </div>
                <CommandList>
                  {diseasesLoading && <div className="p-4 text-center text-sm text-muted-foreground flex items-center justify-center"><Loader2 className="mr-2 h-4 w-4 animate-spin" /> Ачааллаж байна...</div>}
                  {diseasesError && !symptomsError && <div className="p-4 text-center text-sm text-destructive flex items-center justify-center"><AlertCircle className="mr-2 h-4 w-4" /> {diseasesError}</div>}
                  {!diseasesLoading && !diseasesError && (
                    <>
                      <CommandEmpty>{allDiseases.length > 0 ? "Өвчин олдсонгүй." : "Өвчний жагсаалт хоосон байна."}</CommandEmpty>
                      <ScrollArea className="max-h-[150px]">
                        <CommandGroup heading="Боломжит өвчнүүд">
                          {allDiseases // Show all diseases for selection
                            .filter(d => d.disease_name.toLowerCase().includes(diseaseSearchTerm.toLowerCase()))
                            .map((disease) => (
                              <CommandItem
                                key={disease.disease_id}
                                value={disease.disease_name}
                                onSelect={() => handleSelectDisease(disease)}
                                className={cn(
                                  "cursor-pointer flex justify-between items-center",
                                  selectedDisease?.disease_id === disease.disease_id && "bg-accent text-accent-foreground" // Highlight selected
                                )}
                              >
                                <span>{disease.disease_name} {disease.disease_code && `(${disease.disease_code})`}</span>
                                {selectedDisease?.disease_id === disease.disease_id && <Check className="h-4 w-4" />}
                              </CommandItem>
                            ))}
                        </CommandGroup>
                      </ScrollArea>
                    </>
                  )}
                </CommandList>
              </Command>
            </div>
          )}
        </ScrollArea>

        {/* Save Error Area (Unchanged) */}
         {saveError && (
          <div className="mt-2 text-sm text-destructive bg-destructive/10 p-3 rounded-md flex items-center flex-shrink-0">
            <AlertCircle className="mr-2 h-4 w-4 flex-shrink-0" />
            {saveError}
          </div>
        )}

        <DialogFooter className="pt-4 border-t flex-shrink-0">
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={saveLoading}>
            Цуцлах
          </Button>
          <Button
            onClick={handleSave}
            disabled={
              !predictions ||
              selectedSymptoms.length === 0 ||
              !selectedDisease || // Check single selected disease
              saveLoading || predictionLoading || diseasesLoading || symptomsLoading
            }
          >
            {saveLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Хадгалах
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
