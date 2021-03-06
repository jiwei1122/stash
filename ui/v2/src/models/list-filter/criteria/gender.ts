import { CriterionModifier } from "../../../core/generated-graphql";
import { StashService } from "../../../core/StashService";
import {
  Criterion,
  CriterionType,
  ICriterionOption,
} from "./criterion";

export class GenderCriterion extends Criterion<string, string> {
  public type: CriterionType = "gender";
  public parameterName: string = "gender";
  public modifier = CriterionModifier.Equals;
  public modifierOptions = [];
  public options: string[] = StashService.getGenderStrings();
  public value: string = "";
}

export class GenderCriterionOption implements ICriterionOption {
  public label: string = Criterion.getLabel("gender");
  public value: CriterionType = "gender";
}
